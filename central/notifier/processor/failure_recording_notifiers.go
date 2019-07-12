package processor

import (
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
)

// This will attach an expiring cache to the notifier that records all of the alerts it failed to send and allows
// calling a 'retryFailed' function which will retry all of the failures and remove them from the list if they succeed.
func recordFailures(underlying notifiers.Notifier) notifiers.Notifier {
	if ran, ok := underlying.(notifiers.ResolvableAlertNotifier); ok {
		return &failureRecordingResolvableAlertNotifierImpl{
			underlying:   ran,
			failedAlerts: newAlertSet(),
		}
	}
	if an, ok := underlying.(notifiers.AlertNotifier); ok {
		return &failureRecordingAlertNotiferImpl{
			underlying:   an,
			failedAlerts: newAlertSet(),
		}
	}
	return underlying
}

// Interface for an object that records failures.
/////////////////////////////////////////////////

type failureRecorder interface {
	retryFailed()
}

// Record failures for an AlertNotifier.
////////////////////////////////////////

type failureRecordingAlertNotiferImpl struct {
	underlying notifiers.AlertNotifier

	failedAlerts alertSet
}

func (rn *failureRecordingAlertNotiferImpl) ProtoNotifier() *storage.Notifier {
	return rn.underlying.ProtoNotifier()
}

func (rn *failureRecordingAlertNotiferImpl) Test() error {
	return rn.underlying.Test()
}

func (rn *failureRecordingAlertNotiferImpl) AlertNotify(alert *storage.Alert) error {
	err := rn.underlying.AlertNotify(alert)
	if err != nil {
		// If it failed previously, this will simply overwrite the old value.
		rn.failedAlerts.add(alert)
	}
	return err
}

func (rn *failureRecordingAlertNotiferImpl) retryFailed() {
	for _, alert := range rn.failedAlerts.getAll() {
		if err := tryToAlert(rn.underlying, alert); err == nil {
			rn.failedAlerts.remove(alert.GetId())
		}
	}
}

// Record failures for a ResolvableAlertNotifier.
/////////////////////////////////////////////////

type failureRecordingResolvableAlertNotifierImpl struct {
	underlying notifiers.ResolvableAlertNotifier

	failedAlerts alertSet
}

func (rn *failureRecordingResolvableAlertNotifierImpl) ProtoNotifier() *storage.Notifier {
	return rn.underlying.ProtoNotifier()
}

func (rn *failureRecordingResolvableAlertNotifierImpl) Test() error {
	return rn.underlying.Test()
}

func (rn *failureRecordingResolvableAlertNotifierImpl) AlertNotify(alert *storage.Alert) error {
	err := rn.underlying.AlertNotify(alert)
	if err != nil {
		rn.failedAlerts.add(alert)
	}
	return err
}

func (rn *failureRecordingResolvableAlertNotifierImpl) AckAlert(alert *storage.Alert) error {
	err := rn.underlying.AckAlert(alert)
	if err != nil {
		rn.failedAlerts.add(alert)
	}
	return err
}

func (rn *failureRecordingResolvableAlertNotifierImpl) ResolveAlert(alert *storage.Alert) error {
	err := rn.underlying.ResolveAlert(alert)
	if err != nil {
		rn.failedAlerts.add(alert)
	}
	return err
}

func (rn *failureRecordingResolvableAlertNotifierImpl) retryFailed() {
	for _, alert := range rn.failedAlerts.getAll() {
		if err := tryToAlert(rn.underlying, alert); err == nil {
			rn.failedAlerts.remove(alert.GetId())
		}
	}
}
