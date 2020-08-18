package processor

import (
	"context"

	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// Sending alerts.
//////////////////

func tryToAlert(ctx context.Context, notifier notifiers.Notifier, alert *storage.Alert) error {
	if alert.GetState() == storage.ViolationState_ACTIVE {
		alertNotifier, ok := notifier.(notifiers.AlertNotifier)
		if !ok {
			return nil
		}
		return sendNotification(ctx, alertNotifier, alert)
	}

	alertNotifier, ok := notifier.(notifiers.ResolvableAlertNotifier)
	if !ok {
		return nil
	}
	return sendResolvableNotification(alertNotifier, alert)
}

func sendNotification(ctx context.Context, notifier notifiers.AlertNotifier, alert *storage.Alert) error {
	err := notifier.AlertNotify(ctx, alert)
	if err != nil {
		logFailure(notifier, alert, err)
	}
	return err
}

func sendResolvableNotification(notifier notifiers.ResolvableAlertNotifier, alert *storage.Alert) error {
	var err error
	switch alert.GetState() {
	case storage.ViolationState_SNOOZED:
		err = notifier.AckAlert(alert)
	case storage.ViolationState_RESOLVED:
		err = notifier.ResolveAlert(alert)
	}
	if err != nil {
		logFailure(notifier, alert, err)
	}
	return err
}

func logFailure(notifier notifiers.Notifier, alert *storage.Alert, err error) {
	protoNotifier := notifier.ProtoNotifier()
	log.Errorf("Unable to send %s notification to %s (%s) for alert %s: %v", alert.GetState().String(), protoNotifier.GetName(), protoNotifier.GetType(), alert.GetId(), err)
}

// Sending Audit Messages.
//////////////////////////

func tryToSendAudit(ctx context.Context, notifier notifiers.Notifier, msg *v1.Audit_Message) {
	auditNotifier, ok := notifier.(notifiers.AuditNotifier)
	if ok {
		sendAuditMessage(ctx, auditNotifier, msg)
	}
}

func sendAuditMessage(ctx context.Context, notifier notifiers.AuditNotifier, msg *v1.Audit_Message) {
	if err := notifier.SendAuditMessage(ctx, msg); err != nil {
		protoNotifier := notifier.ProtoNotifier()
		log.Errorf("Unable to send audit msg to %s (%s): %v", protoNotifier.GetName(), protoNotifier.GetType(), err)
	}
}
