package processor

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/notifiers"
	"github.com/stackrox/stackrox/central/notifiers/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestNotifierSet(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(notifierSetTestSuite))
}

type notifierSetTestSuite struct {
	suite.Suite

	mockCtrl             *gomock.Controller
	mockAlertN           *mocks.MockAlertNotifier
	mockResolvableAlertN *mocks.MockResolvableAlertNotifier
	mockAuditN           *mocks.MockAuditNotifier

	ns NotifierSet
}

func (s *notifierSetTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.mockAlertN = mocks.NewMockAlertNotifier(s.mockCtrl)
	s.mockResolvableAlertN = mocks.NewMockResolvableAlertNotifier(s.mockCtrl)
	s.mockAuditN = mocks.NewMockAuditNotifier(s.mockCtrl)

	s.ns = NewNotifierSet()
}

func (s *notifierSetTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *notifierSetTestSuite) TestHasFunctions() {
	ctx := context.Background()

	// No notifiers.
	s.False(s.ns.HasNotifiers())
	s.False(s.ns.HasEnabledAuditNotifiers())

	// Only an alert notifier.
	notifier1 := &storage.Notifier{Id: "n1"}
	s.mockAlertN.EXPECT().ProtoNotifier().Return(notifier1)

	s.ns.UpsertNotifier(context.Background(), s.mockAlertN)

	s.True(s.ns.HasNotifiers())
	s.False(s.ns.HasEnabledAuditNotifiers())

	// An alet and an enabled audit notifier.
	notifier2 := &storage.Notifier{Id: "n2"}
	s.mockAuditN.EXPECT().ProtoNotifier().Return(notifier2)
	s.mockAuditN.EXPECT().AuditLoggingEnabled().Return(true)

	s.ns.UpsertNotifier(ctx, s.mockAuditN)

	s.True(s.ns.HasNotifiers())
	s.True(s.ns.HasEnabledAuditNotifiers())
}

func (s *notifierSetTestSuite) TestCoorelatedPoliciesAndNotifiers() {
	ctx := context.Background()

	// Add all of our notifiers.
	notifier1 := &storage.Notifier{Id: "n1"}
	s.mockAlertN.EXPECT().ProtoNotifier().Return(notifier1)
	notifier2 := &storage.Notifier{Id: "n2"}
	s.mockResolvableAlertN.EXPECT().ProtoNotifier().Return(notifier2)
	notifier3 := &storage.Notifier{Id: "n3"}
	s.mockAuditN.EXPECT().ProtoNotifier().Return(notifier3)

	s.ns.UpsertNotifier(ctx, s.mockAlertN)
	s.ns.UpsertNotifier(ctx, s.mockResolvableAlertN)
	s.ns.UpsertNotifier(ctx, s.mockAuditN)

	// Check that the alert notifiers are activated.
	s.mockAlertN.EXPECT().AlertNotify(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.mockResolvableAlertN.EXPECT().AlertNotify(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	s.ns.ForEach(ctx, func(ctx context.Context, n notifiers.Notifier, failures AlertSet) {
		an, ok := n.(notifiers.AlertNotifier)
		if !ok {
			return
		}
		_ = an.AlertNotify(ctx, nil)
	})

	// Check that the resolvable alert notifiers are activated.
	s.mockResolvableAlertN.EXPECT().AlertNotify(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	s.ns.ForEach(ctx, func(ctx context.Context, n notifiers.Notifier, failures AlertSet) {
		an, ok := n.(notifiers.ResolvableAlertNotifier)
		if !ok {
			return
		}
		_ = an.AlertNotify(ctx, nil)
	})

	// Check that the audit notifiers are activated.
	s.mockAuditN.EXPECT().SendAuditMessage(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	s.ns.ForEach(ctx, func(ctx context.Context, n notifiers.Notifier, failures AlertSet) {
		an, ok := n.(notifiers.AuditNotifier)
		if !ok {
			return
		}
		_ = an.SendAuditMessage(ctx, nil)
	})
}
