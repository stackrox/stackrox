package pruning

import (
	"context"
	"time"

	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

const (
	pruneInterval = 24 * time.Hour
)

var (
	log = logging.LoggerForModule()

	pruningCtx = sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.OneStepSCC{
			sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.AllowFixedScopes(
				sac.ResourceScopeKeys(resources.Alert, resources.Config, resources.Deployment, resources.Image)),
			sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.AllowFixedScopes(
				sac.ResourceScopeKeys(resources.Alert, resources.Image)),
		})
)

// GarbageCollector implements a generic garbage collection mechanism
type GarbageCollector interface {
	Start()
	Stop()
}

func newGarbageCollector(alerts alertDatastore.DataStore, images imageDatastore.DataStore, deployments deploymentDatastore.DataStore, config configDatastore.DataStore) GarbageCollector {
	return &garbageCollectorImpl{
		alerts:      alerts,
		images:      images,
		deployments: deployments,
		config:      config,
		stopSig:     concurrency.NewSignal(),
		stoppedSig:  concurrency.NewSignal(),
	}
}

type garbageCollectorImpl struct {
	alerts      alertDatastore.DataStore
	images      imageDatastore.DataStore
	deployments deploymentDatastore.DataStore
	config      configDatastore.DataStore

	stopSig    concurrency.Signal
	stoppedSig concurrency.Signal
}

func (g *garbageCollectorImpl) Start() {
	go g.runGC()
}

func (g *garbageCollectorImpl) runGC() {
	config, err := g.config.GetConfig(pruningCtx)
	if err != nil {
		log.Error(err)
		return
	}
	pvtConfig := config.GetPrivateConfig()
	// Run collection initially then run on a ticker
	g.collectImages(pvtConfig)
	g.collectAlerts(pvtConfig)

	t := time.NewTicker(pruneInterval)
	for {
		select {
		case <-t.C:
			g.collectImages(pvtConfig)
			g.collectAlerts(pvtConfig)
		case <-g.stopSig.Done():
			g.stoppedSig.Signal()
			return
		}
	}
}

func (g *garbageCollectorImpl) collectImages(config *storage.PrivateConfig) {

	pruneImageAfterDays := config.GetImageRetentionDurationDays()
	qb := search.NewQueryBuilder().AddDays(search.LastUpdatedTime, int64(pruneImageAfterDays)).ProtoQuery()
	imageResults, err := g.images.Search(pruningCtx, qb)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("[Image pruning] Found %d image search results", len(imageResults))

	imagesToPrune := make([]string, 0, len(imageResults))
	for _, result := range imageResults {
		q := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, result.ID).ProtoQuery()
		results, err := g.deployments.Search(pruningCtx, q)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Infof("[Image pruning] Found %d search results", len(results))
		// If there are no deployment queries that match, then allow the image to be pruned
		if len(results) == 0 {
			imagesToPrune = append(imagesToPrune, result.ID)
		}
	}
	if len(imagesToPrune) > 0 {
		log.Infof("[Image Pruning] Removing the following images: %+v", imagesToPrune)
		if err := g.images.DeleteImages(pruningCtx, imagesToPrune...); err != nil {
			log.Error(err)
		}
	}
}

func (g *garbageCollectorImpl) collectAlerts(config *storage.PrivateConfig) {

	alertRetention := config.GetAlertRetention()
	if alertRetention == nil {
		log.Infof("[Alert pruning] Alert pruning has been disabled.")
		return
	}

	pruneAlertsAfterDays := config.GetAlertRetentionDurationDays()

	runtimeAlerts := search.NewQueryBuilder().
		AddStrings(search.LifecycleStage, storage.LifecycleStage_RUNTIME.String()).
		AddStrings(search.ViolationState,
			storage.ViolationState_ACTIVE.String(), storage.ViolationState_RESOLVED.String()).
		AddDays(search.ViolationTime, int64(pruneAlertsAfterDays)).ProtoQuery()

	deploytimeAlerts := search.NewQueryBuilder().
		AddStrings(search.LifecycleStage, storage.LifecycleStage_DEPLOY.String()).
		AddStrings(search.ViolationState,
			storage.ViolationState_RESOLVED.String()).
		AddDays(search.ViolationTime, int64(pruneAlertsAfterDays)).ProtoQuery()

	alertResults, err := g.alerts.Search(pruningCtx,
		search.NewDisjunctionQuery([]*v1.Query{runtimeAlerts, deploytimeAlerts}...))

	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("[Alert pruning] Found %d alert search results", len(alertResults))

	alertsToPrune := search.ResultsToIDs(alertResults)
	if len(alertsToPrune) > 0 {
		log.Infof("[Alert pruning] Removing %d alerts", len(alertsToPrune))
		if err := g.alerts.DeleteAlerts(pruningCtx, alertsToPrune...); err != nil {
			log.Error(err)
		}
	}
}

func (g *garbageCollectorImpl) Stop() {
	g.stopSig.Signal()
	<-g.stoppedSig.Done()
}
