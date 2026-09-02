package main

import (
	"context"
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

var listenerCtx = sac.WithAllAccess(context.Background())

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

func (r *reportListener) start(ctx context.Context) {
	listener := pgNotify.NewListener(r.db, r.handleNotification,
		pgNotify.NotifierChanged,
		pgNotify.ReportConfigChanged,
		pgNotify.ReportRequestSubmitted,
		pgNotify.ReportRequestCancelled,
	)
	go listener.Listen(ctx)
	go r.periodicResync(ctx)
}

func (r *reportListener) handleNotification(channel, payload string) {
	switch channel {
	case pgNotify.NotifierChanged:
		r.handleNotifierChanged(payload)
	case pgNotify.ReportConfigChanged:
		r.handleConfigChanged(payload)
	case pgNotify.ReportRequestSubmitted:
		r.handleRequestSubmitted(payload)
	case pgNotify.ReportRequestCancelled:
		r.handleRequestCancelled(payload)
	}
}

func (r *reportListener) handleNotifierChanged(notifierID string) {
	protoNotifier, exists, err := r.notifierStore.GetNotifier(listenerCtx, notifierID)
	if err != nil {
		log.Errorf("Failed to load notifier %s: %v", notifierID, err)
		return
	}
	if !exists {
		r.notifierProcessor.RemoveNotifier(listenerCtx, notifierID)
		return
	}
	n, err := pkgNotifiers.CreateNotifier(protoNotifier)
	if err != nil {
		log.Errorf("Failed to create notifier %s: %v", notifierID, err)
		return
	}
	r.notifierProcessor.UpdateNotifier(listenerCtx, n)
}

func (r *reportListener) handleConfigChanged(configID string) {
	config, exists, err := r.reportConfigStore.GetReportConfiguration(listenerCtx, configID)
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

func (r *reportListener) handleRequestSubmitted(snapshotID string) {
	snap, exists, err := r.snapshotStore.Get(listenerCtx, snapshotID)
	if err != nil || !exists {
		log.Errorf("Failed to load report snapshot %s: exists=%v err=%v", snapshotID, exists, err)
		return
	}
	if snap.GetReportStatus().GetRunState() != storage.ReportStatus_WAITING {
		return
	}

	var collection *storage.ResourceCollection
	if snap.GetCollection().GetId() != "" {
		collection, exists, err = r.collectionDatastore.Get(listenerCtx, snap.GetCollection().GetId())
		if err != nil || !exists {
			log.Errorf("Failed to load collection for snapshot %s: %v", snapshotID, err)
			return
		}
	}

	req := &reportGen.ReportRequest{
		ReportSnapshot: snap,
		Collection:     collection,
	}
	if _, err := r.scheduler.SubmitReportRequest(listenerCtx, req, true); err != nil {
		log.Errorf("Failed to submit report request for snapshot %s: %v", snapshotID, err)
	} else {
		r.trackReportID(snapshotID)
	}
}

func (r *reportListener) handleRequestCancelled(snapshotID string) {
	if _, err := r.scheduler.CancelReportRequest(listenerCtx, snapshotID); err != nil {
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
			r.resyncConfigs()
			r.resyncPendingRequests()
		}
	}
}

func (r *reportListener) resyncConfigs() {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportType,
			storage.ReportConfiguration_VULNERABILITY.String(),
			storage.ReportConfiguration_NODE_VULNERABILITY.String(),
		).
		ProtoQuery()
	configs, err := r.reportConfigStore.GetReportConfigurations(listenerCtx, query)
	if err != nil {
		log.Errorf("Error resyncing report configs: %v", err)
		return
	}

	activeConfigIDs := set.NewStringSet()
	for _, rc := range configs {
		if rc.GetSchedule() != nil && common.HasValidResourceScope(rc.GetResourceScope()) {
			activeConfigIDs.Add(rc.GetId())
			if err := r.scheduler.UpsertReportSchedule(rc); err != nil {
				log.Errorf("Error resyncing schedule for config %s: %v", rc.GetId(), err)
			}
		}
	}

	for _, id := range r.scheduler.GetScheduledConfigIDs() {
		if !activeConfigIDs.Contains(id) {
			r.scheduler.RemoveReportSchedule(id)
		}
	}
}

func (r *reportListener) resyncPendingRequests() {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String()).
		WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.ReportQueuedTime))).
		ProtoQuery()
	snapshots, err := r.snapshotStore.SearchReportSnapshots(listenerCtx, query)
	if err != nil {
		log.Errorf("Error resyncing pending report requests: %v", err)
		return
	}

	for _, snap := range snapshots {
		if r.isReportIDKnown(snap.GetReportId()) {
			continue
		}

		var collection *storage.ResourceCollection
		if collID := snap.GetCollection().GetId(); collID != "" {
			var exists bool
			collection, exists, err = r.collectionDatastore.Get(listenerCtx, collID)
			if err != nil || !exists {
				log.Errorf("Error loading collection for pending snapshot %s: %v", snap.GetReportId(), err)
				continue
			}
		}

		req := &reportGen.ReportRequest{
			ReportSnapshot: snap,
			Collection:     collection,
		}
		if _, err := r.scheduler.SubmitReportRequest(listenerCtx, req, true); err != nil {
			log.Errorf("Error resyncing pending report %s: %v", snap.GetReportId(), err)
		} else {
			r.trackReportID(snap.GetReportId())
		}
	}
}

func (r *reportListener) trackReportID(id string) {
	r.knownMu.Lock()
	defer r.knownMu.Unlock()
	r.knownReportIDs.Add(id)
}

func (r *reportListener) isReportIDKnown(id string) bool {
	r.knownMu.Lock()
	defer r.knownMu.Unlock()
	return r.knownReportIDs.Contains(id)
}
