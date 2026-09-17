package main

import (
	"context"
	"errors"
	"time"

	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/notifier"
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/postgres"
	pgNotify "github.com/stackrox/rox/pkg/postgres/notify"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

type reportListener struct {
	db                  postgres.DB
	scheduler           schedulerV2.Scheduler
	reportConfigStore   reportConfigDS.DataStore
	snapshotStore       reportSnapshotDS.DataStore
	collectionDatastore collectionDS.DataStore
	notifierStore       notifierDS.DataStore
	notifierProcessor   notifier.Processor
	knownMu             sync.Mutex
	knownReportIDs      set.StringSet
	stateMu             sync.Mutex
}

func newReportListener(
	db postgres.DB,
	scheduler schedulerV2.Scheduler,
	reportConfigStore reportConfigDS.DataStore,
	snapshotStore reportSnapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore,
	notifierStore notifierDS.DataStore,
	notifierProcessor notifier.Processor,
) *reportListener {
	return &reportListener{
		db:                  db,
		scheduler:           scheduler,
		reportConfigStore:   reportConfigStore,
		snapshotStore:       snapshotStore,
		collectionDatastore: collectionDatastore,
		notifierStore:       notifierStore,
		notifierProcessor:   notifierProcessor,
		knownReportIDs:      set.NewStringSet(),
	}
}

func (r *reportListener) start(ctx context.Context) func() {
	lifecycleCtx, cancel := context.WithCancel(sac.WithAllAccess(ctx))
	listener := pgNotify.NewListenerWithHooks(r.db, func(channel, payload string) {
		r.handleNotification(lifecycleCtx, channel, payload)
	}, pgNotify.LifecycleHooks{
		AfterListen: func(ctx context.Context) error {
			return r.reconcile(ctx)
		},
		OnDisconnect: func(err error) {
			if err != nil && lifecycleCtx.Err() == nil {
				log.Warnf("Report LISTEN/NOTIFY listener disconnected: %v", err)
			}
		},
	},
		pgNotify.NotifierChanged,
		pgNotify.ReportConfigChanged,
		pgNotify.ReportRequestSubmitted,
		pgNotify.ReportRequestCancelled,
	)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		listener.Listen(lifecycleCtx)
	}()
	go func() {
		defer workers.Done()
		r.periodicResync(lifecycleCtx)
	}()
	return func() {
		cancel()
		workers.Wait()
	}
}

func (r *reportListener) handleNotification(ctx context.Context, channel, payload string) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	switch channel {
	case pgNotify.NotifierChanged:
		r.handleNotifierChanged(ctx, payload)
	case pgNotify.ReportConfigChanged:
		r.handleConfigChanged(ctx, payload)
	case pgNotify.ReportRequestSubmitted:
		r.handleRequestSubmitted(ctx, payload)
	case pgNotify.ReportRequestCancelled:
		r.handleRequestCancelled(ctx, payload)
	}
}

func (r *reportListener) handleNotifierChanged(ctx context.Context, notifierID string) {
	protoNotifier, exists, err := r.notifierStore.GetNotifier(ctx, notifierID)
	if err != nil {
		log.Errorf("Failed to load notifier %s: %v", notifierID, err)
		return
	}
	if !exists {
		r.notifierProcessor.RemoveNotifier(ctx, notifierID)
		return
	}
	if err := r.applyNotifier(ctx, protoNotifier); err != nil {
		log.Errorf("Failed to apply notifier %s: %v", notifierID, err)
		return
	}
}

func (r *reportListener) handleConfigChanged(ctx context.Context, configID string) {
	config, exists, err := r.reportConfigStore.GetReportConfiguration(ctx, configID)
	if err != nil {
		log.Errorf("Failed to load report config %s: %v", configID, err)
		return
	}
	if !exists {
		r.scheduler.RemoveReportSchedule(configID)
		return
	}
	if config.GetSchedule() != nil {
		if err := r.scheduler.UpsertReportSchedule(config); err != nil {
			log.Errorf("Failed to upsert schedule for config %s: %v", configID, err)
		}
	} else {
		r.scheduler.RemoveReportSchedule(configID)
	}
}

func (r *reportListener) handleRequestSubmitted(ctx context.Context, snapshotID string) {
	snap, exists, err := r.snapshotStore.Get(ctx, snapshotID)
	if err != nil || !exists {
		log.Errorf("Failed to load report snapshot %s: exists=%v err=%v", snapshotID, exists, err)
		return
	}
	if snap.GetReportStatus().GetRunState() != storage.ReportStatus_WAITING {
		return
	}

	var collection *storage.ResourceCollection
	if snap.GetCollection().GetId() != "" {
		collection, exists, err = r.collectionDatastore.Get(ctx, snap.GetCollection().GetId())
		if err != nil || !exists {
			log.Errorf("Failed to load collection for snapshot %s: %v", snapshotID, err)
			return
		}
	}

	req := &reportGen.ReportRequest{
		ReportSnapshot: snap,
		Collection:     collection,
	}
	if err := r.submitIfNew(ctx, snapshotID, req); err != nil {
		log.Errorf("Failed to submit report request %s: %v", snapshotID, err)
	}
}

func (r *reportListener) handleRequestCancelled(ctx context.Context, snapshotID string) {
	if _, err := r.scheduler.CancelReportRequest(ctx, snapshotID); err != nil {
		log.Errorf("Failed to cancel report request %s: %v", snapshotID, err)
	}
}

func (r *reportListener) periodicResync(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(env.CentralWorkerResyncIntervalMins.IntegerSetting()) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.reconcile(ctx); err != nil {
				log.Errorf("Error reconciling report worker state: %v", err)
			}
		}
	}
}

func (r *reportListener) reconcile(ctx context.Context) error {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	return errors.Join(
		r.reconcileNotifiers(ctx),
		r.resyncConfigs(ctx),
		r.resyncPendingRequests(ctx),
	)
}

func (r *reportListener) reconcileNotifiers(ctx context.Context) error {
	protoNotifiers := make(map[string]*storage.Notifier)
	if err := r.notifierStore.ForEachNotifier(ctx, func(protoNotifier *storage.Notifier) error {
		if protoNotifier != nil {
			protoNotifiers[protoNotifier.GetId()] = protoNotifier.CloneVT()
		}
		return nil
	}); err != nil {
		return err
	}

	loadedNotifiers := make(map[string]pkgNotifiers.Notifier)
	for _, loaded := range r.notifierProcessor.GetNotifiers(ctx) {
		if loaded != nil && loaded.ProtoNotifier() != nil {
			loadedNotifiers[loaded.ProtoNotifier().GetId()] = loaded
		}
	}

	created := make(map[string]pkgNotifiers.Notifier)
	for id, protoNotifier := range protoNotifiers {
		loaded := loadedNotifiers[id]
		if loaded != nil && loaded.ProtoNotifier().EqualVT(protoNotifier) {
			continue
		}
		n, err := pkgNotifiers.CreateNotifier(protoNotifier.CloneVT())
		if err != nil {
			for _, createdNotifier := range created {
				if closeErr := createdNotifier.Close(ctx); closeErr != nil {
					log.Warnf("Failed to close notifier %s after incomplete reconciliation: %v", createdNotifier.ProtoNotifier().GetId(), closeErr)
				}
			}
			return errors.Join(err, errors.New("notifier reconciliation is incomplete; retaining current state"))
		}
		created[id] = n
	}

	for id, n := range created {
		r.notifierProcessor.UpdateNotifier(ctx, n)
		delete(loadedNotifiers, id)
	}
	for id := range protoNotifiers {
		delete(loadedNotifiers, id)
	}
	for id := range loadedNotifiers {
		r.notifierProcessor.RemoveNotifier(ctx, id)
	}
	return nil
}

func (r *reportListener) applyNotifier(ctx context.Context, protoNotifier *storage.Notifier) error {
	if loaded := r.notifierProcessor.GetNotifier(ctx, protoNotifier.GetId()); loaded != nil &&
		loaded.ProtoNotifier().EqualVT(protoNotifier) {
		return nil
	}
	n, err := pkgNotifiers.CreateNotifier(protoNotifier.CloneVT())
	if err != nil {
		return err
	}
	r.notifierProcessor.UpdateNotifier(ctx, n)
	return nil
}

func (r *reportListener) resyncConfigs(ctx context.Context) error {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportType,
			storage.ReportConfiguration_VULNERABILITY.String(),
			storage.ReportConfiguration_NODE_VULNERABILITY.String(),
		).
		ProtoQuery()
	configs, err := r.reportConfigStore.GetReportConfigurations(ctx, query)
	if err != nil {
		return err
	}

	activeConfigIDs := set.NewStringSet()
	for _, rc := range configs {
		if rc.GetSchedule() != nil && common.HasValidResourceScope(rc.GetResourceScope()) {
			activeConfigIDs.Add(rc.GetId())
			if err := r.scheduler.UpsertReportSchedule(rc); err != nil {
				return err
			}
		}
	}

	for _, id := range r.scheduler.GetScheduledConfigIDs() {
		if !activeConfigIDs.Contains(id) {
			r.scheduler.RemoveReportSchedule(id)
		}
	}
	return nil
}

func (r *reportListener) resyncPendingRequests(ctx context.Context) error {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String()).
		WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ReportQueuedTime))).
		ProtoQuery()
	snapshots, err := r.snapshotStore.SearchReportSnapshots(ctx, query)
	if err != nil {
		return err
	}

	var reconciliationErr error
	for _, snap := range snapshots {
		var collection *storage.ResourceCollection
		if collID := snap.GetCollection().GetId(); collID != "" {
			var exists bool
			collection, exists, err = r.collectionDatastore.Get(ctx, collID)
			if err != nil || !exists {
				log.Errorf("Error loading collection for pending snapshot %s: %v", snap.GetReportId(), err)
				if err == nil {
					err = errors.New("collection does not exist")
				}
				reconciliationErr = errors.Join(reconciliationErr, err)
				continue
			}
		}

		req := &reportGen.ReportRequest{
			ReportSnapshot: snap,
			Collection:     collection,
		}
		if err := r.submitIfNew(ctx, snap.GetReportId(), req); err != nil {
			reconciliationErr = errors.Join(reconciliationErr, err)
		}
	}
	return reconciliationErr
}

func (r *reportListener) submitIfNew(ctx context.Context, reportID string, req *reportGen.ReportRequest) error {
	r.knownMu.Lock()
	defer r.knownMu.Unlock()

	if r.knownReportIDs.Contains(reportID) {
		return nil
	}
	if _, err := r.scheduler.SubmitReportRequest(ctx, req, true); err != nil {
		return err
	}
	r.knownReportIDs.Add(reportID)
	return nil
}
