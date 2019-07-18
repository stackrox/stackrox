package processor

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	notifierMocks "github.com/stackrox/rox/central/notifiers/mocks"
	"github.com/stackrox/rox/generated/storage"
)

func TestProcessor_LoopDoesNothing(t *testing.T) {
	// Create mocks.
	mockCtrl := gomock.NewController(t)

	alertNotfierProto := &storage.Notifier{Id: "n1"}
	mockAlertNotifier := notifierMocks.NewMockAlertNotifier(mockCtrl)

	resolvableAlertNotfierProto := &storage.Notifier{Id: "n2"}
	mockResolvableNotifier := notifierMocks.NewMockResolvableAlertNotifier(mockCtrl)

	// Create our tested objects.
	ns := NewNotifierSet()
	processor := &processorImpl{ns: ns}
	loop := &loopImpl{ns: ns}

	// Add the notifiers to the processor.
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto)
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	processor.UpdateNotifier(mockAlertNotifier)
	processor.UpdateNotifier(mockResolvableNotifier)

	// Retry previous failures. (None)
	loop.retryFailures()
	mockCtrl.Finish()
}

func TestProcessor_LoopDoesNothingIfAllSucceed(t *testing.T) {
	// Create mocks.
	mockCtrl := gomock.NewController(t)

	alertNotfierProto := &storage.Notifier{Id: "n1"}
	mockAlertNotifier := notifierMocks.NewMockAlertNotifier(mockCtrl)

	resolvableAlertNotfierProto := &storage.Notifier{Id: "n2"}
	mockResolvableNotifier := notifierMocks.NewMockResolvableAlertNotifier(mockCtrl)

	policy := &storage.Policy{
		Id:        "p1",
		Notifiers: []string{"n1", "n2"},
	}

	// Create our tested objects.
	ns := NewNotifierSet()
	processor := &processorImpl{ns: ns}
	loop := &loopImpl{ns: ns}

	// Add the notifiers to the processor. (Called once on insert, and once for each alert processed)
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto).Times(3)
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto).Times(3)

	processor.UpdateNotifier(mockAlertNotifier)
	processor.UpdateNotifier(mockResolvableNotifier)

	// Running the loop should do anything if all of the alerts succeed.
	activeAlert := &storage.Alert{
		Id:     "a1",
		State:  storage.ViolationState_ACTIVE,
		Policy: policy,
	}
	mockAlertNotifier.EXPECT().AlertNotify(activeAlert).Return(nil)
	mockResolvableNotifier.EXPECT().AlertNotify(activeAlert).Return(nil)

	snoozedAlert := &storage.Alert{
		Id:     "a2",
		State:  storage.ViolationState_SNOOZED,
		Policy: policy,
	}
	mockResolvableNotifier.EXPECT().AckAlert(snoozedAlert).Return(nil)

	processor.processAlertSync(activeAlert)
	processor.processAlertSync(snoozedAlert)

	// Retry previous failures. (None)
	loop.retryFailures()
	mockCtrl.Finish()
}

func TestProcessor_LoopHandlesFailures(t *testing.T) {
	// Create mocks.
	mockCtrl := gomock.NewController(t)

	alertNotfierProto := &storage.Notifier{Id: "n1"}
	mockAlertNotifier := notifierMocks.NewMockAlertNotifier(mockCtrl)

	resolvableAlertNotfierProto := &storage.Notifier{Id: "n2"}
	mockResolvableNotifier := notifierMocks.NewMockResolvableAlertNotifier(mockCtrl)

	policy := &storage.Policy{
		Id:        "p1",
		Notifiers: []string{"n1", "n2"},
	}

	// Create our tested objects.
	ns := NewNotifierSet()
	processor := &processorImpl{ns: ns}
	loop := &loopImpl{ns: ns}

	// Add the notifiers to the processor. (Called once on insert, and once for each alert processed)
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto).Times(3)
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto).Times(3)

	processor.UpdateNotifier(mockAlertNotifier)
	processor.UpdateNotifier(mockResolvableNotifier)

	// Running the loop should do anything if all of the alerts succeed.
	activeAlert := &storage.Alert{
		Id:     "a1",
		State:  storage.ViolationState_ACTIVE,
		Policy: policy,
	}
	mockAlertNotifier.EXPECT().AlertNotify(activeAlert).Return(errors.New("broke"))
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto)
	mockResolvableNotifier.EXPECT().AlertNotify(activeAlert).Return(errors.New("broke"))
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	snoozedAlert := &storage.Alert{
		Id:     "a2",
		State:  storage.ViolationState_SNOOZED,
		Policy: policy,
	}
	mockResolvableNotifier.EXPECT().AckAlert(snoozedAlert).Return(errors.New("broke"))
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	processor.processAlertSync(activeAlert)
	processor.processAlertSync(snoozedAlert)

	// Retry previous failures. (All of the calls)
	mockAlertNotifier.EXPECT().AlertNotify(activeAlert).Return(nil)
	mockResolvableNotifier.EXPECT().AlertNotify(activeAlert).Return(nil)

	mockResolvableNotifier.EXPECT().AckAlert(snoozedAlert).Return(errors.New("broke"))
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	loop.retryFailures()

	// Retry previous failures. (Just the ack on the snoozed)
	mockResolvableNotifier.EXPECT().AckAlert(snoozedAlert).Return(nil)

	loop.retryFailures()

	// Retry previous failures. (None)
	loop.retryFailures()
	mockCtrl.Finish()
}
