package microsoftsentinel

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = logging.LoggerForModule(option.EnableAdministrationEvents())
)

var _ notifiers.AlertNotifier = (*sentinel)(nil)
var _ notifiers.AuditNotifier = (*sentinel)(nil)

//var _ notifiers.NetworkPolicyNotifier = (*sentinel)(nil)

func init() {
	notifiers.Add(notifiers.MicrosoftSentinelType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
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
		return nil, errors.Wrap(err, "could not create azure credentials")
	}

	client, err := azlogs.NewClient(config.GetLogIngestionEndpoint(), azureCredentials, &azlogs.ClientOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not create azure logs client")
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
	//marhsaler := jsonpb.Marshaler{}
	//jsonString, err := marhsaler.MarshalToString(msg)
	//fmt.Println("err", err)
	//fmt.Println(string(jsonString))
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
	alert := &storage.Alert{
		ClusterId:   "cluster-id",
		ClusterName: "cluster-01",
		Namespace:   "default",
		NamespaceId: "default-ns-id",
		Policy: &storage.Policy{
			Id:              "policy-id",
			Name:            "some-policy",
			Categories:      make([]string, 0),
			Description:     "",
			Remediation:     "",
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD, storage.LifecycleStage_DEPLOY},
		},
		//Entity: &storage.Alert_Deployment_{},
		//Violations: []*storage.Alert_Violation{
		//	{Type: storage.Alert_Violation_GENERIC, Message: "", Time: timestamppb.Now()},
		//},
		//ProcessViolation: &storage.Alert_ProcessViolation{},
		//Enforcement: &storage.Alert_Enforcement{
		//	Action: storage.EnforcementAction_KILL_POD_ENFORCEMENT,
		//},
		Time:          timestamppb.Now(),
		FirstOccurred: timestamppb.Now(),
		ResolvedAt:    timestamppb.Now(),
		State:         storage.ViolationState_ACTIVE,
		SnoozeTill:    timestamppb.Now(),
	}
	if err := s.AlertNotify(ctx, alert); err != nil {
		return notifiers.NewNotifierError("could not send event", err)
	}
	return nil
}

func (s sentinel) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	log.Info("Called AlertNotify Sentinel")

	marshaler := jsonpb.Marshaler{}
	buffer := bytes.NewBuffer([]byte{})
	err := marshaler.Marshal(buffer, alert)
	if err != nil {
		return errors.Wrap(err, "failed to convert alert to json")
	}

	out := map[string]interface{}{}
	err = json.Unmarshal(buffer.Bytes(), &out)
	if err != nil {
		return errors.Wrap(err, "failed to convert to map")
	}

	// 2024-08-20T11:29:23.5949298Z
	// 2024-08-20T11:33:52.0430238Z

	// 2024-08-20T11:29:22.605675619Z
	// 2006-01-02T15:04:05.9999999Z07:00
	out["resolvedAt2"] = alert.GetResolvedAt().AsTime().Format(time.RFC3339Nano)

	// ISO 8601
	out["violationTime2"] = alert.GetTime().AsTime().Format("2006-01-02T15:04:05.99999Z")
	out["firstOccurredTime"] = alert.GetFirstOccurred().AsTime().Format("2006-01-02T15:04:05.99999Z")

	delete(out, "time")
	delete(out, "lifecycleStages")

	outSlice := []map[string]interface{}{out}

	data, err := json.Marshal(outSlice)
	if err != nil {
		return errors.Wrap(err, "failed to convert to binary")
	}
	log.Infof("Alert Print: %+v", string(data))

	log.Infof("Configuration %+v", s.sentinel())

	// UploadResponse is unhandled because it currently is only a placeholder in the azure client library and does not
	// contain any information to be processed.
	_, err = s.azlogsClient.Upload(ctx, s.sentinel().GetDataCollectionRuleId(), s.sentinel().GetStreamName(), data, &azlogs.UploadOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to upload logs to azure")
	}

	return nil
}

func Validate(sentinel *storage.MicrosoftSentinel, validateSecret bool) error {
	errorList := errorhelpers.NewErrorList("Microsoft Sentinel validation")
	if sentinel.GetLogIngestionEndpoint() == "" {
		errorList.AddString("Log Ingestion Endpoint must be specified")
	}

	if sentinel.GetDataCollectionRuleId() == "" {
		errorList.AddString("Data Collection Rule Id must be specified")
	}

	if sentinel.GetStreamName() == "" {
		errorList.AddString("Stream Name must be specified")
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

	return nil
}
