package microsoftsentinel

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
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
		return newSentinelNotifier(notifier), nil
	})
}

type sentinel struct {
	notifier *storage.Notifier
}

func newSentinelNotifier(notifier *storage.Notifier) *sentinel {
	log.Info("Added sentinel notifier")
	return &sentinel{
		notifier: notifier,
	}
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
	log.Info("Called AlertNotify")
	marhsaler := jsonpb.Marshaler{}
	jsonString, err := marhsaler.MarshalToString(alert)
	fmt.Println("err", err)
	fmt.Println(string(jsonString))
	return nil
}
