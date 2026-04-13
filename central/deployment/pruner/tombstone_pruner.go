package pruner

import (
	"context"
	"sync/atomic"
	"time"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log = logging.LoggerForModule()
	// Pruning interval: how often to check for expired deployments.
	pruneInterval = env.PruneInterval.DurationSetting()
	pruningCtx    = sac.WithAllAccess(context.Background())
)

// TombstonePruner removes expired deployment tombstones.
//
//go:generate mockgen-wrapper
type TombstonePruner interface {
	Start()
	Stop()
}

type tombstonePrunerImpl struct {
	deployments deploymentDatastore.DataStore
	stopper     concurrency.Stopper

	// Metrics.
	lastPruneTime time.Time
	prunedTotal   atomic.Uint64
}

// NewTombstonePruner creates a new tombstone pruner.
func NewTombstonePruner(deployments deploymentDatastore.DataStore) TombstonePruner {
	return &tombstonePrunerImpl{
		deployments:   deployments,
		stopper:       concurrency.NewStopper(),
		lastPruneTime: time.Time{},
	}
}

// Start begins the background pruning goroutine.
func (p *tombstonePrunerImpl) Start() {
	go p.runPruning()
}

// Stop gracefully stops the pruning goroutine.
func (p *tombstonePrunerImpl) Stop() {
	p.stopper.Client().Stop()
	_ = p.stopper.Client().Stopped().Wait()
}

// runPruning is the main pruning loop.
func (p *tombstonePrunerImpl) runPruning() {
	defer p.stopper.Flow().ReportStopped()

	// Run initial pruning cycle immediately on startup.
	p.pruneExpiredDeployments()

	ticker := time.NewTicker(pruneInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.pruneExpiredDeployments()
		case <-p.stopper.Flow().StopRequested():
			return
		}
	}
}

// pruneExpiredDeployments queries for expired deployments and hard deletes them.
func (p *tombstonePrunerImpl) pruneExpiredDeployments() {
	defer metrics.SetPruningDuration(time.Now(), "DeploymentTombstones")
	startTime := time.Now()

	// Get all deployments with tombstone.expires_at < now.
	expiredDeployments, err := p.deployments.GetExpiredDeployments(pruningCtx)
	if err != nil {
		log.Errorf("[Deployment Tombstone Pruning] Error finding expired deployments: %v", err)
		return
	}

	if len(expiredDeployments) == 0 {
		log.Debug("[Deployment Tombstone Pruning] No expired deployments to prune")
		p.lastPruneTime = startTime
		return
	}

	log.Infof("[Deployment Tombstone Pruning] Found %d expired deployments. Deleting...", len(expiredDeployments))

	pruned := 0
	for _, deployment := range expiredDeployments {
		// Hard delete: call RemoveDeployment.
		// This will permanently remove the deployment from the database.
		if err := p.deployments.RemoveDeployment(pruningCtx, deployment.GetClusterId(), deployment.GetId()); err != nil {
			log.Errorf("[Deployment Tombstone Pruning] Failed to remove deployment %s: %v", deployment.GetId(), err)
			continue
		}
		pruned++
	}

	p.prunedTotal.Add(uint64(pruned))
	p.lastPruneTime = startTime

	log.Infof("[Deployment Tombstone Pruning] Pruned %d expired deployments (total: %d)", pruned, p.prunedTotal.Load())
}

// GetLastPruneTime returns the time of the last successful prune cycle (for testing/monitoring).
func (p *tombstonePrunerImpl) GetLastPruneTime() time.Time {
	return p.lastPruneTime
}

// GetPrunedTotal returns the total number of deployments pruned (for testing/monitoring).
func (p *tombstonePrunerImpl) GetPrunedTotal() uint64 {
	return p.prunedTotal.Load()
}
