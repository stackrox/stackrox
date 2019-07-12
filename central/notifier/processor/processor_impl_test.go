package processor

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/notifiers"
	notifierMocks "github.com/stackrox/rox/central/notifiers/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestProcessor_RecordersAreCastable(t *testing.T) {
	// Create mocks.
	mockCtrl := gomock.NewController(t)
	mockAlertNotifier := notifierMocks.NewMockAlertNotifier(mockCtrl)
	mockResolvableNotifier := notifierMocks.NewMockResolvableAlertNotifier(mockCtrl)

	recordingAlertNotifier := recordFailures(mockAlertNotifier)
	_, ok := recordingAlertNotifier.(notifiers.ResolvableAlertNotifier)
	assert.False(t, ok)
	_, ok = recordingAlertNotifier.(notifiers.AlertNotifier)
	assert.True(t, ok)

	recordingResolvableNotifier := recordFailures(mockResolvableNotifier)
	_, ok = recordingResolvableNotifier.(notifiers.ResolvableAlertNotifier)
	assert.True(t, ok)
	_, ok = recordingResolvableNotifier.(notifiers.AlertNotifier)
	assert.True(t, ok)
}

func TestProcessor_LoopDoesNothing(t *testing.T) {
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
	pns := newPolicyNotifierSet()
	processor := &processorImpl{pns: pns}
	loop := &loopImpl{pns: pns}

	// Add the notifiers to the processor.
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto)
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	processor.UpdatePolicy(policy)
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
	pns := newPolicyNotifierSet()
	processor := &processorImpl{pns: pns}
	loop := &loopImpl{pns: pns}

	// Add the notifiers to the processor.
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto)
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	processor.UpdateNotifier(mockAlertNotifier)
	processor.UpdateNotifier(mockResolvableNotifier)
	processor.UpdatePolicy(policy)

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
	pns := newPolicyNotifierSet()
	processor := &processorImpl{pns: pns}
	loop := &loopImpl{pns: pns}

	// Add the notifiers to the processor.
	mockAlertNotifier.EXPECT().ProtoNotifier().Return(alertNotfierProto)
	mockResolvableNotifier.EXPECT().ProtoNotifier().Return(resolvableAlertNotfierProto)

	processor.UpdatePolicy(policy)
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
