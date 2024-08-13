package microsoftsentinel

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
)

var (
	log = logging.LoggerForModule(option.EnableAdministrationEvents())
)

var _ notifiers.AlertNotifier = (*sentinel)(nil)
var _ notifiers.AuditNotifier = (*sentinel)(nil)

func init() {
	notifiers.Add("microsoft_sentinel", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		return newSentinelNotifier(notifier)
	})
}

type sentinel struct {
	notifier     *storage.Notifier
	azlogsClient *azlogs.Client
	config       *storage.Notifier
}

func newSentinelNotifier(notifier *storage.Notifier) (*sentinel, error) {
	log.Info("Added sentinel notifier")
	config := notifier.GetMicrosoftSentinel()

	// TODO: Support certificate authentication
	azureCredentials, err := azidentity.NewClientSecretCredential(config.GetDirectoryTenantId(), config.GetApplicationClientId(), config.GetSecret(), &azidentity.ClientSecretCredentialOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not create sentinel client")
	}

	client, err := azlogs.NewClient(config.GetLogIngestionEndpoint(), azureCredentials, &azlogs.ClientOptions{})
	if err != nil {
		log.Fatal(err)
	}

	return &sentinel{
		notifier:     notifier,
		azlogsClient: client,
		config:       notifier,
	}, nil
}

func (s sentinel) sentinel() *storage.MicrosoftSentinel {
	return s.config.GetMicrosoftSentinel()
}

func (s sentinel) SendAuditMessage(ctx context.Context, msg *v1.Audit_Message) error {
	log.Info("Called SendAuditMessage")
	marhsaler := jsonpb.Marshaler{}
	jsonString, err := marhsaler.MarshalToString(msg)
	fmt.Println("err", err)
	fmt.Println(string(jsonString))
	return nil
}

func (s sentinel) AuditLoggingEnabled() bool {
	return true
}

func (s sentinel) Close(ctx context.Context) error {
	log.Info("Called Close")
	return nil
}

func (s sentinel) ProtoNotifier() *storage.Notifier {
	return s.notifier
}

func (s sentinel) Test(ctx context.Context) *notifiers.NotifierError {
	log.Info("Called Test")
	return nil
}

func (s sentinel) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	log.Info("Called AlertNotify Sentinel")

	var alertToUpload []byte
	marshaler := jsonpb.Marshaler{}
	buffer := bytes.NewBuffer(alertToUpload)
	err := marshaler.Marshal(buffer, alert)
	if err != nil {
		return errors.Wrap(err, "failed to convert alert to json")
	}

	// UploadResponse is unhandled because it currently is only a placeholder in the azure client library and does not
	// contain any information to be processed.
	_, err = s.azlogsClient.Upload(ctx, s.sentinel().GetDataCollectionRuleId(), s.sentinel().GetStreamName(), alertToUpload, &azlogs.UploadOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to upload logs to azure")
	}
	return nil
}
