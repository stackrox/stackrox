package processor

import (
	"context"

	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
)

// Sending alerts.
//////////////////

func tryToAlert(ctx context.Context, notifier notifiers.Notifier, alert *storage.Alert) error {
	if alert.GetState() == storage.ViolationState_ACTIVE || alert.GetState() == storage.ViolationState_ATTEMPTED {
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
		err = notifier.AckAlert(context.Background(), alert)
	case storage.ViolationState_RESOLVED:
		err = notifier.ResolveAlert(context.Background(), alert)
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
