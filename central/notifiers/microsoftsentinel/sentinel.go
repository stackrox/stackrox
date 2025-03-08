package microsoftsentinel

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"time"

	azureErrors "github.com/Azure/azure-sdk-for-go-extensions/pkg/errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"github.com/pkg/errors"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/x509utils"
	"google.golang.org/protobuf/proto"
)

var (
	log = logging.LoggerForModule(option.EnableAdministrationEvents())

	_ notifiers.AlertNotifier = (*sentinel)(nil)
	_ notifiers.AuditNotifier = (*sentinel)(nil)
)

func init() {
	if features.MicrosoftSentinelNotifier.Enabled() {
		cryptoKey := ""
		var err error
		if env.EncNotifierCreds.BooleanSetting() {
			cryptoKey, _, err = notifierUtils.GetActiveNotifierEncryptionKey()
			if err != nil {
				utils.Should(errors.Wrap(err, "Error reading encryption key, notifier will be unable to send notifications"))
			}
		}

		log.Debug("Microsoft Sentinel notifier enabled.")
		notifiers.Add(notifiers.MicrosoftSentinelType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
			return newSentinelNotifier(notifier, cryptocodec.Singleton(), cryptoKey)
		})
	}
}

type sentinel struct {
	notifier     *storage.Notifier
	azlogsClient azureLogsClient
}

func (s sentinel) SendAuditMessage(ctx context.Context, msg *v1.Audit_Message) error {
	if !features.MicrosoftSentinelNotifier.Enabled() {
		return nil
	}

	if !s.AuditLoggingEnabled() {
		return nil
	}

	err := s.uploadLogs(ctx, s.notifier.GetMicrosoftSentinel().GetAuditLogDcrConfig(), msg)
	if err != nil {
		return errors.Wrap(err, "failed to upload audit log to Microsoft Sentinel")
	}
	return nil
}

func (s sentinel) AuditLoggingEnabled() bool {
	return s.notifier.GetMicrosoftSentinel().GetAuditLogDcrConfig().GetEnabled()
}

// newSentinelNotifier returns a new sentinel notifier.
func newSentinelNotifier(notifier *storage.Notifier, cryptoCodec cryptocodec.CryptoCodec, key string) (*sentinel, error) {
	config := notifier.GetMicrosoftSentinel()

	err := Validate(config, false)
	if err != nil {
		return nil, errors.Wrap(err, "could not create sentinel notifier, validation failed")
	}

	secret, err := notifierUtils.GetCredentials(notifier)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not read secret for notifier %q", notifier.GetName())
	}

	if env.EncNotifierCreds.BooleanSetting() {
		// Decrypted secret is a secret when using secret auth or the private key belonging to a
		// client certificate used in client cert authentication.
		secret, err = cryptoCodec.Decrypt(key, notifier.GetNotifierSecret())
		if err != nil {
			return nil, errors.Errorf("Error decrypting notifier secret for notifier %q", notifier.GetName())
		}
	}

	// Tries to build authentication token in the following order:
	// 1. Azure default credential chain if workload identity is enabled.
	// 2. Client certificate authentication.
	// 3. Static client secret credentials.
	var azureTokenCredential azcore.TokenCredential
	authErrList := errorhelpers.NewErrorList("Sentinel authentication")
	if config.GetWifEnabled() {
		credOpts := &azidentity.DefaultAzureCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: &http.Client{Transport: proxy.RoundTripper()},
			},
		}
		azureTokenCredential, err = azidentity.NewDefaultAzureCredential(credOpts)
		if err != nil {
			return nil, errors.Wrap(err, "obtaining Azure default credentials")
		}
	}

	if config.GetClientCertAuthConfig().GetClientCert() != "" && azureTokenCredential == nil {
		certs, err := x509utils.ConvertPEMTox509Certs([]byte(config.GetClientCertAuthConfig().GetClientCert()))
		if err != nil {
			return nil, errors.Wrap(err, "invalid cert")
		}

		keyBlock, rest := pem.Decode([]byte(secret))
		if len(rest) != 0 {
			log.Errorf("PEM was not valid, could not parse all data: %s", string(rest))
			return nil, errors.Errorf("PEM was not valid, could not parse all data: %s", string(rest))
		}
		if keyBlock == nil {
			return nil, errors.New("could not parse empty key")
		}
		privateKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse azure sentinel private key")
		}

		credOpts := &azidentity.ClientCertificateCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: &http.Client{Transport: proxy.RoundTripper()},
			},
		}
		azureTokenCredential, err = azidentity.NewClientCertificateCredential(
			config.GetDirectoryTenantId(),
			config.GetApplicationClientId(),
			certs,
			privateKey,
			credOpts,
		)
		if err != nil {
			authErrList.AddError(errors.Wrap(err, "could not create azure sentinel credentials with client cert"))
		}
	}

	// If client cert authentication is not configured use secret auth.
	if config.GetClientCertAuthConfig().GetClientCert() == "" && azureTokenCredential == nil {
		var err error
		credOpts := &azidentity.ClientSecretCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Transport: &http.Client{Transport: proxy.RoundTripper()},
			},
		}
		azureTokenCredential, err = azidentity.NewClientSecretCredential(
			config.GetDirectoryTenantId(),
			config.GetApplicationClientId(),
			secret,
			credOpts,
		)
		if err != nil {
			authErrList.AddError(errors.Wrap(err, "could not create azure credentials with secret"))
		}
	}

	// If no token was created then authentication failed.
	if azureTokenCredential == nil {
		return nil, authErrList
	}

	client, err := azlogs.NewClient(config.GetLogIngestionEndpoint(), azureTokenCredential, &azlogs.ClientOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not create azure logs client")
	}

	return &sentinel{
		notifier:     notifier,
		azlogsClient: &azureLogsClientImpl{client: client},
	}, nil
}

func (s sentinel) Close(_ context.Context) error {
	return nil
}

func (s sentinel) ProtoNotifier() *storage.Notifier {
	return s.notifier
}

func (s sentinel) Test(ctx context.Context) *notifiers.NotifierError {
	if s.notifier.GetMicrosoftSentinel().GetAuditLogDcrConfig().GetEnabled() {
		err := s.SendAuditMessage(ctx, s.getTestAuditLogMessage())
		if err != nil {
			return notifiers.NewNotifierError("could not send audit message to sentinel", err)
		}
	} else {
		log.Info("audit message are disabled, test audit message was not send to sentinel")
	}

	if s.notifier.GetMicrosoftSentinel().GetAlertDcrConfig().GetEnabled() {
		err := s.AlertNotify(ctx, s.getTestAlert())
		if err != nil {
			return notifiers.NewNotifierError("could not send alert notify to sentinel", err)
		}
	} else {
		log.Info("alert notifier is disabled, test alert was not send to sentinel")
	}

	return nil
}

func (s sentinel) getTestAuditLogMessage() *v1.Audit_Message {
	return &v1.Audit_Message{
		Request: &v1.Audit_Message_Request{
			Endpoint: "test-endpoint",
			Method:   "GET",
		},
	}
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

	err := s.uploadLogs(ctx, s.notifier.GetMicrosoftSentinel().GetAlertDcrConfig(), alert)
	if err != nil {
		return errors.Wrap(err, "failed to upload alert notifications to Microsoft Sentinel")
	}
	return nil
}

func (s sentinel) uploadLogs(ctx context.Context, dcrConfig *storage.MicrosoftSentinel_DataCollectionRuleConfig, msg proto.Message) error {
	bytesToSend, err := s.prepareLogsToSend(msg)
	if err != nil {
		return err
	}

	return retry.WithRetry(func() error {
		// UploadResponse is unhandled because it currently is only a placeholder in the azure client library and does not
		// contain any information to be processed.
		_, err := s.azlogsClient.Upload(ctx, dcrConfig.GetDataCollectionRuleId(), dcrConfig.GetStreamName(), bytesToSend, &azlogs.UploadOptions{})
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
}

// prepareLogsToSend converts a proto message, wraps it into an array and converts it to JSON which is expected by Sentinel.
func (s sentinel) prepareLogsToSend(msg protocompat.Message) ([]byte, error) {
	// convert object to an unstructured map to later wrap it as an array.
	logToSendObj, err := protocompat.MarshalMap(msg)
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

	if sentinel.GetAuditLogDcrConfig().GetEnabled() {
		if sentinel.GetAuditLogDcrConfig().GetDataCollectionRuleId() == "" {
			errorList.AddString("Audit Logging Data Collection Rule Id must be specified")
		}

		if sentinel.GetAuditLogDcrConfig().GetStreamName() == "" {
			errorList.AddString("Audit Logging Stream Name must be specified")
		}
	}

	if sentinel.GetAlertDcrConfig().GetEnabled() {
		if sentinel.GetAlertDcrConfig().GetDataCollectionRuleId() == "" {
			errorList.AddString("Alert Data Collection Rule Id must be specified")
		}

		if sentinel.GetAlertDcrConfig().GetStreamName() == "" {
			errorList.AddString("Alert Stream Name must be specified")
		}
	}

	if sentinel.GetDirectoryTenantId() == "" {
		errorList.AddString("Directory Tenant Id must be specified")
	}

	if !sentinel.GetWifEnabled() && sentinel.GetApplicationClientId() == "" {
		errorList.AddString("Application Client Id must be specified")
	}

	if (!sentinel.GetWifEnabled() && sentinel.GetSecret() == "" && (sentinel.GetClientCertAuthConfig().GetClientCert() == "" || sentinel.GetClientCertAuthConfig().GetPrivateKey() == "")) && validateSecret {
		errorList.AddString("Secret, Client Certificate or Workload Identity authentication must be specified")
	}

	if !errorList.Empty() {
		return errorList
	}
	return nil
}
