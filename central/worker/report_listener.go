package main

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	pgNotify "github.com/stackrox/rox/pkg/postgres/notify"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

const configResyncInterval = 5 * time.Minute

var listenerCtx = sac.WithAllAccess(context.Background())

type reportListener struct {
	db                  postgres.DB
	scheduler           schedulerV2.Scheduler
	reportConfigStore   reportConfigDS.DataStore
	snapshotStore       reportSnapshotDS.DataStore
	collectionDatastore collectionDS.DataStore
}

func newReportListener(
	db postgres.DB,
	scheduler schedulerV2.Scheduler,
	reportConfigStore reportConfigDS.DataStore,
	snapshotStore reportSnapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore,
) *reportListener {
	return &reportListener{
		db:                  db,
		scheduler:           scheduler,
		reportConfigStore:   reportConfigStore,
		snapshotStore:       snapshotStore,
		collectionDatastore: collectionDatastore,
	}
}

func (r *reportListener) start(ctx context.Context) {
	listener := pgNotify.NewListener(r.db, r.handleNotification,
		pgNotify.ReportConfigChanged,
		pgNotify.ReportRequestSubmitted,
		pgNotify.ReportRequestCancelled,
	)
	go listener.Listen(ctx)
	go r.periodicResync(ctx)
}

func (r *reportListener) handleNotification(channel, payload string) {
	switch channel {
	case pgNotify.ReportConfigChanged:
		r.handleConfigChanged(payload)
	case pgNotify.ReportRequestSubmitted:
		r.handleRequestSubmitted(payload)
	case pgNotify.ReportRequestCancelled:
		r.handleRequestCancelled(payload)
	}
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
	}
}

func (r *reportListener) handleRequestCancelled(snapshotID string) {
	if _, err := r.scheduler.CancelReportRequest(listenerCtx, snapshotID); err != nil {
		log.Errorf("Failed to cancel report request %s: %v", snapshotID, err)
	}
}

func (r *reportListener) periodicResync(ctx context.Context) {
	ticker := time.NewTicker(configResyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.resyncConfigs()
		}
	}
}

func (r *reportListener) resyncConfigs() {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportType, storage.ReportConfiguration_VULNERABILITY.String()).
		ProtoQuery()
	configs, err := r.reportConfigStore.GetReportConfigurations(listenerCtx, query)
	if err != nil {
		log.Errorf("Error resyncing report configs: %v", err)
		return
	}
	for _, rc := range configs {
		if rc.GetSchedule() != nil && common.HasValidResourceScope(rc.GetResourceScope()) {
			if err := r.scheduler.UpsertReportSchedule(rc); err != nil {
				log.Errorf("Error resyncing schedule for config %s: %v", rc.GetId(), err)
			}
		}
	}
}
