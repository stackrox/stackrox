package microsoftsentinel

import (
	"context"
	"encoding/json"
	"time"

	azureErrors "github.com/Azure/azure-sdk-for-go-extensions/pkg/errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule(option.EnableAdministrationEvents())

	_ notifiers.AlertNotifier = (*sentinel)(nil)
)

func init() {
	if features.MicrosoftSentinelNotifier.Enabled() {
		log.Debug("Microsoft Sentinel notifier enabled.")
		notifiers.Add(notifiers.MicrosoftSentinelType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
			return newSentinelNotifier(notifier)
		})
	}
}

type sentinel struct {
	notifier     *storage.Notifier
	azlogsClient azureLogsClient
}

func newSentinelNotifier(notifier *storage.Notifier) (*sentinel, error) {
	config := notifier.GetMicrosoftSentinel()

	err := Validate(config, false)
	if err != nil {
		return nil, errors.Wrap(err, "could not create sentinel notifier, validation failed")
	}

	// TODO(ROX-25739): Support certificate authentication
	azureCredentials, err := azidentity.NewClientSecretCredential(config.GetDirectoryTenantId(), config.GetApplicationClientId(), config.GetSecret(), &azidentity.ClientSecretCredentialOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not create azure credentials")
	}

	client, err := azlogs.NewClient(config.GetLogIngestionEndpoint(), azureCredentials, &azlogs.ClientOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not create azure logs client")
	}

	return &sentinel{
		notifier:     notifier,
		azlogsClient: &azureLogsClientImpl{client: client},
	}, nil
}

func (s sentinel) sentinel() *storage.MicrosoftSentinel {
	return s.notifier.GetMicrosoftSentinel()
}

func (s sentinel) Close(_ context.Context) error {
	return nil
}

func (s sentinel) ProtoNotifier() *storage.Notifier {
	return s.notifier
}

func (s sentinel) Test(ctx context.Context) *notifiers.NotifierError {
	alert := s.getTestAlert()

	err := s.AlertNotify(ctx, alert)
	if err != nil {
		return notifiers.NewNotifierError("could not send event", err)
	}
	return nil
}

func (s sentinel) getTestAlert() *storage.Alert {
	alert := &storage.Alert{
		Policy: &storage.Policy{
			Name:        "test-policy",
			Description: "Test policy description",
		},
		ClusterName: "test-cluster",
		ClusterId:   uuid.NewDummy().String(),
		Namespace:   "test-namespace",
	}
	return alert
}

func (s sentinel) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	if !features.MicrosoftSentinelNotifier.Enabled() {
		return errors.New("Microsoft Sentinel notifier is disabled.")
	}

	if !s.notifier.GetMicrosoftSentinel().GetAlertDcrConfig().GetEnabled() {
		return nil
	}

	bytesToSend, err := s.prepareLogsToSend(alert)
	if err != nil {
		return err
	}

	err = retry.WithRetry(func() error {
		// UploadResponse is unhandled because it currently is only a placeholder in the azure client library and does not
		// contain any information to be processed.
		_, err := s.azlogsClient.Upload(ctx, s.sentinel().GetAlertDcrConfig().GetDataCollectionRuleId(), s.sentinel().GetAlertDcrConfig().GetStreamName(), bytesToSend, &azlogs.UploadOptions{})
		azRespErr := azureErrors.IsResponseError(err)
		if azRespErr != nil {
			return notifiers.CreateError(s.notifier.GetName(), azRespErr.RawResponse, codes.MicrosoftSentinelGeneric)
		}
		return err
	},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)

	if err != nil {
		return errors.Wrap(err, "failed to upload logs to azure")
	}
	return nil
}

func (s sentinel) prepareLogsToSend(alert *storage.Alert) ([]byte, error) {
	// convert object to an unstructured map to later wrap it as an array.
	logToSendObj, err := protocompat.MarshalMap(alert)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send alert to Microsoft Sentinel")
	}

	// Wrap object in a slice because Sentinel expects it.
	logsToSend := []map[string]interface{}{
		{"msg": logToSendObj},
	}

	bytesToSend, err := json.Marshal(logsToSend)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap alert for Microsoft Sentinel to a slice")
	}

	return bytesToSend, nil
}

// Validate validates a Microsoft Sentinel configuration.
func Validate(sentinel *storage.MicrosoftSentinel, validateSecret bool) error {
	if !features.MicrosoftSentinelNotifier.Enabled() {
		return errors.New("Microsoft Sentinel notifier is disabled.")
	}

	errorList := errorhelpers.NewErrorList("Microsoft Sentinel validation")
	if sentinel.GetLogIngestionEndpoint() == "" {
		errorList.AddString("Log Ingestion Endpoint must be specified")
	}

	if sentinel.GetAlertDcrConfig().GetEnabled() {
		if sentinel.GetAlertDcrConfig().GetDataCollectionRuleId() == "" {
			errorList.AddString("Data Collection Rule Id must be specified")
		}

		if sentinel.GetAlertDcrConfig().GetStreamName() == "" {
			errorList.AddString("Stream Name must be specified")
		}
	}

	if sentinel.GetDirectoryTenantId() == "" {
		errorList.AddString("Directory Tenant Id must be specified")
	}

	if sentinel.GetApplicationClientId() == "" {
		errorList.AddString("Application Client Id must be specified")
	}

	if sentinel.GetSecret() == "" && validateSecret {
		errorList.AddString("Secret must be specified")
	}

	if !errorList.Empty() {
		return errorList
	}
	return nil
}
