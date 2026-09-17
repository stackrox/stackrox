//go:build sql_integration

package main

import (
	"context"
	"testing"
	"time"

	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	configMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	schedulerMocks "github.com/stackrox/rox/central/reports/scheduler/v2/mocks"
	snapshotMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	collectionMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"go.uber.org/mock/gomock"
)

func TestReportListenerStartReconcilesBeforeShutdown(t *testing.T) {
	testDB := pgtest.ForT(t)
	ctrl := gomock.NewController(t)
	notifierStore := notifierDSMocks.NewMockDataStore(ctrl)
	configStore := configMocks.NewMockDataStore(ctrl)
	snapshotStore := snapshotMocks.NewMockDataStore(ctrl)
	collectionStore := collectionMocks.NewMockDataStore(ctrl)
	scheduler := schedulerMocks.NewMockScheduler(ctrl)
	processor := &testProcessor{loaded: map[string]notifiers.Notifier{}}

	reconciled := make(chan struct{})
	notifierStore.EXPECT().ForEachNotifier(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ func(*storage.Notifier) error) error {
		close(reconciled)
		return nil
	})
	configStore.EXPECT().GetReportConfigurations(gomock.Any(), gomock.Any()).Return(nil, nil)
	scheduler.EXPECT().GetScheduledConfigIDs().Return(nil)
	snapshotStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return(nil, nil)

	listener := newReportListener(testDB.DB, scheduler, configStore, snapshotStore, collectionStore, notifierStore, processor)
	stop := listener.start(context.Background())

	select {
	case <-reconciled:
	case <-time.After(5 * time.Second):
		t.Fatal("report listener did not reconcile after LISTEN setup")
	}
	stop()
}
