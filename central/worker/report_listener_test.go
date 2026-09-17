package main

import (
	"context"
	"errors"
	"testing"
	"time"

	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testNotifierType = "central-worker-report-listener-test"

type testReportNotifier struct {
	proto  *storage.Notifier
	closed int
}

func (n *testReportNotifier) Close(context.Context) error {
	n.closed++
	return nil
}

func (n *testReportNotifier) ProtoNotifier() *storage.Notifier {
	return n.proto
}

func (n *testReportNotifier) Test(context.Context) *notifiers.NotifierError {
	return nil
}

func registerTestNotifier(t *testing.T, create func(*storage.Notifier) (notifiers.Notifier, error)) {
	t.Helper()
	previous, wasRegistered := notifiers.Registry[testNotifierType]
	notifiers.Add(testNotifierType, create)
	t.Cleanup(func() {
		if wasRegistered {
			notifiers.Registry[testNotifierType] = previous
		} else {
			delete(notifiers.Registry, testNotifierType)
		}
	})
}

type testProcessor struct {
	loaded map[string]notifiers.Notifier
}

func (p *testProcessor) ProcessAlert(context.Context, *storage.Alert)           {}
func (p *testProcessor) ProcessAuditMessage(context.Context, *v1.Audit_Message) {}
func (p *testProcessor) HasNotifiers() bool                                     { return len(p.loaded) > 0 }
func (p *testProcessor) HasEnabledAuditNotifiers() bool                         { return false }
func (p *testProcessor) UpdateNotifier(_ context.Context, n notifiers.Notifier) {
	p.loaded[n.ProtoNotifier().GetId()] = n
}
func (p *testProcessor) RemoveNotifier(ctx context.Context, id string) {
	if loaded := p.loaded[id]; loaded != nil {
		_ = loaded.Close(ctx)
	}
	delete(p.loaded, id)
}
func (p *testProcessor) GetNotifier(_ context.Context, id string) notifiers.Notifier {
	return p.loaded[id]
}
func (p *testProcessor) GetNotifiers(context.Context) []notifiers.Notifier {
	result := make([]notifiers.Notifier, 0, len(p.loaded))
	for _, n := range p.loaded {
		result = append(result, n)
	}
	return result
}
func (p *testProcessor) UpdateNotifierHealthStatus(notifiers.Notifier, storage.IntegrationHealth_Status, string) {
}

func newTestReportListener(t *testing.T, processor *testProcessor) (*reportListener, *notifierDSMocks.MockDataStore) {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := notifierDSMocks.NewMockDataStore(ctrl)
	return &reportListener{notifierStore: store, notifierProcessor: processor}, store
}

func TestReportListenerReconcileNotifiersPreservesUnchangedAndRemovesDeleted(t *testing.T) {
	keep := &storage.Notifier{Id: "keep", Type: testNotifierType, Name: "unchanged"}
	stale := &storage.Notifier{Id: "stale", Type: testNotifierType, Name: "deleted"}
	unchanged := &testReportNotifier{proto: keep.CloneVT()}
	staleNotifier := &testReportNotifier{proto: stale.CloneVT()}
	processor := &testProcessor{loaded: map[string]notifiers.Notifier{
		keep.GetId():  unchanged,
		stale.GetId(): staleNotifier,
	}}
	listener, store := newTestReportListener(t, processor)
	creatorCalls := 0
	registerTestNotifier(t, func(proto *storage.Notifier) (notifiers.Notifier, error) {
		creatorCalls++
		return &testReportNotifier{proto: proto.CloneVT()}, nil
	})
	store.EXPECT().ForEachNotifier(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(*storage.Notifier) error) error {
		return fn(keep.CloneVT())
	})

	require.NoError(t, listener.reconcileNotifiers(context.Background()))
	require.Same(t, unchanged, processor.loaded[keep.GetId()])
	_, staleStillLoaded := processor.loaded[stale.GetId()]
	require.False(t, staleStillLoaded)
	require.Equal(t, 0, creatorCalls)
	require.Equal(t, 0, unchanged.closed)
	require.Equal(t, 1, staleNotifier.closed)
}

func TestReportListenerReconcileNotifiersDoesNotApplyPartialEnumeration(t *testing.T) {
	loaded := &storage.Notifier{Id: "loaded", Type: testNotifierType}
	loadedNotifier := &testReportNotifier{proto: loaded.CloneVT()}
	processor := &testProcessor{loaded: map[string]notifiers.Notifier{loaded.GetId(): loadedNotifier}}
	listener, store := newTestReportListener(t, processor)
	enumerationErr := errors.New("database walk failed")
	store.EXPECT().ForEachNotifier(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(*storage.Notifier) error) error {
		require.NoError(t, fn(loaded.CloneVT()))
		return enumerationErr
	})

	require.ErrorIs(t, listener.reconcileNotifiers(context.Background()), enumerationErr)
	_, stillLoaded := processor.loaded[loaded.GetId()]
	require.True(t, stillLoaded)
	require.Equal(t, 0, loadedNotifier.closed)
}

func TestReportListenerReconcileNotifiersDoesNotApplyFailedConstruction(t *testing.T) {
	loaded := &storage.Notifier{Id: "loaded", Type: testNotifierType, Name: "old"}
	updated := loaded.CloneVT()
	updated.Name = "new"
	loadedNotifier := &testReportNotifier{proto: loaded.CloneVT()}
	processor := &testProcessor{loaded: map[string]notifiers.Notifier{loaded.GetId(): loadedNotifier}}
	listener, store := newTestReportListener(t, processor)
	constructionErr := errors.New("creator failed")
	registerTestNotifier(t, func(*storage.Notifier) (notifiers.Notifier, error) {
		return nil, constructionErr
	})
	store.EXPECT().ForEachNotifier(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(*storage.Notifier) error) error {
		return fn(updated)
	})

	require.ErrorIs(t, listener.reconcileNotifiers(context.Background()), constructionErr)
	require.Same(t, loadedNotifier, processor.loaded[loaded.GetId()])
	require.Equal(t, 0, loadedNotifier.closed)
}

func TestReportListenerReconcileNotifiersClosesCreatedInstancesOnFailure(t *testing.T) {
	first := &storage.Notifier{Id: "first", Type: testNotifierType}
	second := &storage.Notifier{Id: "second", Type: testNotifierType}
	processor := &testProcessor{loaded: map[string]notifiers.Notifier{}}
	listener, store := newTestReportListener(t, processor)
	var created []*testReportNotifier
	constructionErr := errors.New("creator failed")
	registerTestNotifier(t, func(proto *storage.Notifier) (notifiers.Notifier, error) {
		if len(created) == 1 {
			return nil, constructionErr
		}
		n := &testReportNotifier{proto: proto.CloneVT()}
		created = append(created, n)
		return n, nil
	})
	store.EXPECT().ForEachNotifier(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(*storage.Notifier) error) error {
		require.NoError(t, fn(first.CloneVT()))
		return fn(second.CloneVT())
	})

	require.ErrorIs(t, listener.reconcileNotifiers(context.Background()), constructionErr)
	require.Len(t, created, 1)
	require.Equal(t, 1, created[0].closed)
	require.Empty(t, processor.loaded)
}

func TestReportListenerPeriodicResyncStopsWithLifecycleContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		(&reportListener{}).periodicResync(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("periodic reconciliation did not stop after context cancellation")
	}
}
