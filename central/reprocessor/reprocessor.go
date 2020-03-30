package reprocessor

import (
	"context"
	"time"

	"github.com/pkg/errors"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/throttle"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

var (
	log = logging.LoggerForModule()

	riskDedupeNamespace = uuid.NewV4()

	once sync.Once
	loop Loop

	maxInjectionDelay = 500 * time.Millisecond

	getDeploymentsContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment)))

	getAndWriteImagesContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Image)))
)

// Singleton returns the singleton reprocessor loop
func Singleton() Loop {
	once.Do(func() {
		loop = NewLoop(connection.ManagerSingleton(), enrichment.ImageEnricherSingleton(), deploymentDatastore.Singleton(), imageDatastore.Singleton())
	})
	return loop
}

// Loop combines periodically (every hour) runs enrichment and detection.
//go:generate mockgen-wrapper
type Loop interface {
	Start()
	ShortCircuit()
	Stop()

	ReprocessRisk()
	ReprocessRiskForDeployments(deploymentIDs ...string)
}

// NewLoop returns a new instance of a Loop.
func NewLoop(connManager connection.Manager, enricher enricher.ImageEnricher, deployments deploymentDatastore.DataStore, images imageDatastore.DataStore) Loop {
	return newLoopWithDuration(connManager, enricher, deployments, images, env.ReprocessInterval.DurationSetting(), 30*time.Minute, 15*time.Second)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(connManager connection.Manager, enricher enricher.ImageEnricher, deployments deploymentDatastore.DataStore, images imageDatastore.DataStore, enrichAndDetectDuration,
	enrichAndDetectInjectionPeriod, deploymentRiskDuration time.Duration) Loop {
	return &loopImpl{
		enrichAndDetectTickerDuration:  enrichAndDetectDuration,
		deploymenRiskTickerDuration:    deploymentRiskDuration,
		enrichAndDetectInjectionPeriod: enrichAndDetectInjectionPeriod,

		enricher: enricher,
		images:   images,

		deployments:       deployments,
		deploymentRiskSet: set.NewStringSet(),

		stopChan:  concurrency.NewSignal(),
		stopped:   concurrency.NewSignal(),
		shortChan: make(chan struct{}),

		connManager: connManager,
		throttler:   throttle.NewDropThrottle(time.Second),
	}
}

type loopImpl struct {
	enrichAndDetectTickerDuration  time.Duration
	enrichAndDetectInjectionPeriod time.Duration
	enrichAndDetectTicker          *time.Ticker

	images   imageDatastore.DataStore
	enricher enricher.ImageEnricher

	deployments                 deploymentDatastore.DataStore
	deploymentRiskSet           set.StringSet
	deploymentRiskLock          sync.Mutex
	deploymentRiskTicker        *time.Ticker
	deploymenRiskTickerDuration time.Duration

	shortChan chan struct{}
	stopChan  concurrency.Signal
	stopped   concurrency.Signal

	connManager connection.Manager
	throttler   throttle.DropThrottle
}

func (l *loopImpl) ReprocessRisk() {
	l.throttler.Run(func() { l.sendDeployments(true, 0) })
}

func (l *loopImpl) ReprocessRiskForDeployments(deploymentIDs ...string) {
	l.deploymentRiskLock.Lock()
	defer l.deploymentRiskLock.Unlock()
	l.deploymentRiskSet.AddAll(deploymentIDs...)
}

// Start starts the enrich and detect loop.
func (l *loopImpl) Start() {
	l.enrichAndDetectTicker = time.NewTicker(l.enrichAndDetectTickerDuration)
	l.deploymentRiskTicker = time.NewTicker(l.deploymenRiskTickerDuration)
	go l.loop()
}

// Stop stops the enrich and detect loop.
func (l *loopImpl) Stop() {
	l.stopChan.Signal()
	l.stopped.Wait()
}

func (l *loopImpl) ShortCircuit() {
	go l.reprocessImagesAndResyncDeployments(enricher.UseCachesIfPossible)
}

func (l *loopImpl) sendDeployments(riskOnly bool, injectionPeriod time.Duration, deploymentIDs ...string) {
	query := search.NewQueryBuilder().AddStringsHighlighted(search.ClusterID, search.WildcardString)
	if len(deploymentIDs) > 0 {
		query = query.AddDocIDs(deploymentIDs...)
	}

	results, err := l.deployments.SearchDeployments(getDeploymentsContext, query.ProtoQuery())
	if err != nil {
		log.Errorf("error getting results for reprocessing: %v", err)
		return
	}

	path, ok := deployments.OptionsMap.Get(search.ClusterID.String())
	if !ok {
		panic("No Cluster ID option for deployments")
	}

	var injectionLimiter *rate.Limiter
	if injectionPeriod != 0 && len(results) != 0 {
		calculatedRate := time.Duration(l.enrichAndDetectInjectionPeriod.Nanoseconds() / int64(len(results)))
		if calculatedRate > maxInjectionDelay {
			calculatedRate = maxInjectionDelay
		}
		injectionLimiter = rate.NewLimiter(rate.Every(calculatedRate), 1)
	}

	for _, r := range results {
		clusterIDs := r.FieldToMatches[path.FieldPath].GetValues()
		if len(clusterIDs) == 0 {
			log.Error("no cluster id found in fields")
			continue
		}

		conn := l.connManager.GetConnection(clusterIDs[0])
		if conn == nil {
			continue
		}

		var dedupeKey string
		if riskOnly {
			dedupeKey = uuid.NewV5(riskDedupeNamespace, r.Id).String()
		}
		if injectionLimiter != nil {
			_ = injectionLimiter.Wait(context.Background())
		}

		msg := &central.MsgFromSensor{
			HashKey:   r.Id,
			DedupeKey: dedupeKey,
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Resource: &central.SensorEvent_ReprocessDeployment{
						ReprocessDeployment: &central.ReprocessDeploymentRisk{
							DeploymentId: r.Id,
						},
					},
				},
			},
		}

		conn.InjectMessageIntoQueue(msg)
	}
}

func (l *loopImpl) reprocessImage(id string, sema *semaphore.Weighted, wg *sync.WaitGroup, fetchOpts enricher.FetchOption) {
	defer sema.Release(1)
	defer wg.Done()

	image, exists, err := l.images.GetImage(getAndWriteImagesContext, id, false)
	if err != nil {
		log.Errorf("error fetching image %q from the database: %v", id, err)
		return
	}
	if !exists {
		return
	}

	result, err := l.enricher.EnrichImage(enricher.EnrichmentContext{
		FetchOpt: fetchOpts,
	}, image)

	if err != nil {
		log.Errorf("error enriching image: %v", err)
		return
	}
	if result.ImageUpdated {
		if err := l.images.UpsertImage(getAndWriteImagesContext, image); err != nil {
			log.Errorf("error upserting image %q into datastore: %v", image.GetName().GetFullName(), err)
		}
	}
}

func (l *loopImpl) getActiveImageIDs() ([]string, error) {
	query := search.NewQueryBuilder().AddStringsHighlighted(search.DeploymentID, search.WildcardString).ProtoQuery()
	results, err := l.images.Search(getAndWriteImagesContext, query)
	if err != nil {
		return nil, errors.Wrap(err, "error searching for active image IDs")
	}

	imagesSet := set.NewStringSet()
	for _, result := range results {
		imagesSet.Add(result.ID)
	}
	return imagesSet.AsSlice(), nil
}

func (l *loopImpl) reprocessImagesAndResyncDeployments(fetchOpts enricher.FetchOption) {
	if l.stopped.IsDone() {
		return
	}
	imageIDs, err := l.getActiveImageIDs()
	if err != nil {
		log.Errorf("error retrieving active image ids: %v", err)
		return
	}
	log.Infof("Found %d images to rescan", len(imageIDs))
	imageSemaphore := semaphore.NewWeighted(5)
	var wg sync.WaitGroup
	for _, imageID := range imageIDs {
		wg.Add(1)
		// Go over the image IDs
		if err := imageSemaphore.Acquire(concurrency.AsContext(&l.stopChan), 1); err != nil {
			log.Errorf("context cancelled via stop: %v", err)
			return
		}
		go l.reprocessImage(imageID, imageSemaphore, &wg, fetchOpts)
	}
	wg.Wait()
	log.Infof("Successfully reprocessed %d images. Reevaluating deployments...", len(imageIDs))
	// Once the images have been rescanned, then reprocess the images
	// This should not take a particular long period of time
	if !l.stopped.IsDone() {
		l.connManager.BroadcastMessage(&central.MsgToSensor{
			Msg: &central.MsgToSensor_ReassessPolicies{
				ReassessPolicies: &central.ReassessPolicies{},
			},
		})
	}
}

func (l *loopImpl) loop() {
	defer l.stopped.Signal()
	defer l.enrichAndDetectTicker.Stop()
	defer l.deploymentRiskTicker.Stop()

	// Call reprocessImagesAndResyncDeployments on start to try and ensure that the deployments are processed
	// in <= the reprocessing interval
	go l.reprocessImagesAndResyncDeployments(enricher.ForceRefetch)
	for {
		select {
		case <-l.stopChan.Done():
			return
		case <-l.shortChan:
			l.sendDeployments(false, 0)
		case <-l.enrichAndDetectTicker.C:
			l.reprocessImagesAndResyncDeployments(enricher.ForceRefetchScansOnly)
		case <-l.deploymentRiskTicker.C:
			l.deploymentRiskLock.Lock()
			if l.deploymentRiskSet.Cardinality() > 0 {
				l.sendDeployments(true, 0, l.deploymentRiskSet.AsSlice()...)
				l.deploymentRiskSet.Clear()
			}
			l.deploymentRiskLock.Unlock()
		}
	}
}
