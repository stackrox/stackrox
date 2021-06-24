package processor

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	notifierMocks "github.com/stackrox/rox/central/notifiers/mocks"
	"github.com/stackrox/rox/generated/storage"
)

func TestProcessor_LoopDoesNothing(t *testing.T) {
	ctx := context.Background()
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

	processor.UpdateNotifier(ctx, mockAlertNotifier)
	processor.UpdateNotifier(ctx, mockResolvableNotifier)

	// Retry previous failures. (None)
	loop.retryFailures(ctx)
	mockCtrl.Finish()
}

func TestProcessor_LoopDoesNothingIfAllSucceed(t *testing.T) {
	ctx := context.Background()
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
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto).Times(4)
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto).Times(4)

	processor.UpdateNotifier(ctx, mockAlertNotifier)
	processor.UpdateNotifier(ctx, mockResolvableNotifier)

	// Running the loop should do anything if all of the alerts succeed.
	activeAlert := &storage.Alert{
		Id:     "a1",
		State:  storage.ViolationState_ACTIVE,
		Policy: policy,
	}
	mockAlertNotifier.EXPECT().AlertNotify(gomock.Any(), activeAlert).Return(nil)
	mockResolvableNotifier.EXPECT().AlertNotify(ctx, activeAlert).Return(nil)

	snoozedAlert := &storage.Alert{
		Id:     "a2",
		State:  storage.ViolationState_SNOOZED,
		Policy: policy,
	}
	mockResolvableNotifier.EXPECT().AckAlert(context.Background(), snoozedAlert).Return(nil)

	attemptedAlert := &storage.Alert{
		Id:     "a3",
		State:  storage.ViolationState_ATTEMPTED,
		Policy: policy,
	}
	mockAlertNotifier.EXPECT().AlertNotify(gomock.Any(), attemptedAlert).Return(nil)
	mockResolvableNotifier.EXPECT().AlertNotify(ctx, attemptedAlert).Return(nil)

	processor.processAlertSync(ctx, activeAlert)
	processor.processAlertSync(ctx, snoozedAlert)
	processor.processAlertSync(ctx, attemptedAlert)

	// Retry previous failures. (None)
	loop.retryFailures(ctx)
	mockCtrl.Finish()
}

func TestProcessor_LoopHandlesFailures(t *testing.T) {
	ctx := context.Background()
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
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto).Times(4)
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto).Times(4)

	processor.UpdateNotifier(ctx, mockAlertNotifier)
	processor.UpdateNotifier(ctx, mockResolvableNotifier)

	// Running the loop should do anything if all of the alerts succeed.
	activeAlert := &storage.Alert{
		Id:     "a1",
		State:  storage.ViolationState_ACTIVE,
		Policy: policy,
	}
	mockAlertNotifier.EXPECT().AlertNotify(gomock.Any(), activeAlert).Return(errors.New("broke"))
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto)
	mockResolvableNotifier.EXPECT().AlertNotify(gomock.Any(), activeAlert).Return(errors.New("broke"))
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	snoozedAlert := &storage.Alert{
		Id:     "a2",
		State:  storage.ViolationState_SNOOZED,
		Policy: policy,
	}
	mockResolvableNotifier.EXPECT().AckAlert(context.Background(), snoozedAlert).Return(errors.New("broke"))
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	attemptedAlert := &storage.Alert{
		Id:     "a3",
		State:  storage.ViolationState_ATTEMPTED,
		Policy: policy,
	}
	mockAlertNotifier.EXPECT().AlertNotify(gomock.Any(), attemptedAlert).Return(errors.New("broke"))
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto)
	mockResolvableNotifier.EXPECT().AlertNotify(gomock.Any(), attemptedAlert).Return(errors.New("broke"))
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	processor.processAlertSync(ctx, activeAlert)
	processor.processAlertSync(ctx, snoozedAlert)
	processor.processAlertSync(ctx, attemptedAlert)

	// Retry previous failures. (All of the calls)
	mockAlertNotifier.EXPECT().AlertNotify(gomock.Any(), activeAlert).Return(nil)
	mockResolvableNotifier.EXPECT().AlertNotify(ctx, activeAlert).Return(nil)

	mockResolvableNotifier.EXPECT().AckAlert(context.Background(), snoozedAlert).Return(errors.New("broke"))
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	mockAlertNotifier.EXPECT().AlertNotify(gomock.Any(), attemptedAlert).Return(nil)
	mockResolvableNotifier.EXPECT().AlertNotify(ctx, attemptedAlert).Return(nil)

	loop.retryFailures(ctx)

	// Retry previous failures. (Just the ack on the snoozed)
	mockResolvableNotifier.EXPECT().AckAlert(context.Background(), snoozedAlert).Return(nil)

	loop.retryFailures(ctx)

	// Retry previous failures. (None)
	loop.retryFailures(ctx)
	mockCtrl.Finish()
}
