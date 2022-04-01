package reprocessor

import (
	"context"
	"time"

	"github.com/pkg/errors"
	activeComponentsUpdater "github.com/stackrox/rox/central/activecomponent/updater"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	nodeEnricher "github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	imageMapping "github.com/stackrox/rox/pkg/search/options/images"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/atomic"
	"golang.org/x/sync/semaphore"
)

var (
	log = logging.LoggerForModule()

	riskDedupeNamespace = uuid.NewV4()

	once sync.Once
	loop Loop

	allAccessCtx = sac.WithAllAccess(context.Background())

	emptyCtx = context.Background()

	imageClusterIDFieldPath = imageMapping.ImageDeploymentOptions.MustGet(search.ClusterID.String()).GetFieldPath()

	allImagesQuery = search.NewQueryBuilder().AddStringsHighlighted(search.ClusterID, search.WildcardString).
			ProtoQuery()

	imagesWithSignatureVerificationResultsQuery = search.NewQueryBuilder().
							AddStringsHighlighted(search.ClusterID, search.WildcardString).
							AddStrings(search.ImageSignatureFetchedTime, search.WildcardString).ProtoQuery()
)

// Singleton returns the singleton reprocessor loop
func Singleton() Loop {
	once.Do(func() {
		loop = NewLoop(connection.ManagerSingleton(), enrichment.ImageEnricherSingleton(), enrichment.NodeEnricherSingleton(),
			deploymentDatastore.Singleton(), imageDatastore.Singleton(), nodeDatastore.Singleton(), manager.Singleton(),
			watchedImageDataStore.Singleton(), activeComponentsUpdater.Singleton(), dackbox.GetIndexQueue())
	})
	return loop
}

// Loop combines periodically (every 4 hours by default) runs enrichment and detection.
//go:generate mockgen-wrapper
type Loop interface {
	Start()
	ShortCircuit()
	Stop()

	ReprocessRiskForDeployments(deploymentIDs ...string)
	ReprocessSignatureVerifications()
}

// NewLoop returns a new instance of a Loop.
func NewLoop(connManager connection.Manager, imageEnricher imageEnricher.ImageEnricher, nodeEnricher nodeEnricher.NodeEnricher,
	deployments deploymentDatastore.DataStore, images imageDatastore.DataStore, nodes nodeDatastore.DataStore,
	risk manager.Manager, watchedImages watchedImageDataStore.DataStore, acUpdater activeComponentsUpdater.Updater, indexQueue queue.WaitableQueue) Loop {
	return newLoopWithDuration(
		connManager, imageEnricher, nodeEnricher, deployments, images, nodes, risk, watchedImages,
		env.ReprocessInterval.DurationSetting(), 15*time.Second, env.ActiveVulnRefreshInterval.DurationSetting(), acUpdater, indexQueue)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(connManager connection.Manager, imageEnricher imageEnricher.ImageEnricher, nodeEnricher nodeEnricher.NodeEnricher,
	deployments deploymentDatastore.DataStore, images imageDatastore.DataStore, nodes nodeDatastore.DataStore,
	risk manager.Manager, watchedImages watchedImageDataStore.DataStore, enrichAndDetectDuration, deploymentRiskDuration time.Duration,
	activeComponentTickerDuration time.Duration, acUpdater activeComponentsUpdater.Updater, indexQueue queue.WaitableQueue) *loopImpl {
	return &loopImpl{
		enrichAndDetectTickerDuration: enrichAndDetectDuration,
		deploymentRiskTickerDuration:  deploymentRiskDuration,

		imageEnricher: imageEnricher,
		images:        images,
		risk:          risk,

		watchedImages: watchedImages,

		deployments:       deployments,
		deploymentRiskSet: set.NewStringSet(),

		activeComponentTickerDuration: activeComponentTickerDuration,
		activeComponentStopped:        concurrency.NewSignal(),
		acUpdater:                     acUpdater,

		nodeEnricher: nodeEnricher,
		nodes:        nodes,

		shortCircuitSig:   concurrency.NewSignal(),
		stopSig:           concurrency.NewSignal(),
		enrichmentStopped: concurrency.NewSignal(),
		riskStopped:       concurrency.NewSignal(),

		signatureVerificationSig: concurrency.NewSignal(),

		// Used for testing purposes
		reprocessingStarted:  concurrency.NewSignal(),
		reprocessingComplete: concurrency.NewSignal(),

		connManager: connManager,
		indexQueue:  indexQueue,
	}
}

// imageReprocessingFunc represents the function used for image reprocessing. This enables us to specifically exclude
// some parts of the enrichment, i.e. when only wanting to re-fetch signature verification results.
type imageReprocessingFunc func(ctx context.Context, enrichCtx imageEnricher.EnrichmentContext,
	image *storage.Image) (imageEnricher.EnrichmentResult, error)

type loopImpl struct {
	enrichAndDetectTickerDuration time.Duration
	enrichAndDetectTicker         *time.Ticker

	images        imageDatastore.DataStore
	risk          manager.Manager
	imageEnricher imageEnricher.ImageEnricher

	watchedImages watchedImageDataStore.DataStore

	deployments                  deploymentDatastore.DataStore
	deploymentRiskSet            set.StringSet
	deploymentRiskLock           sync.Mutex
	deploymentRiskTicker         *time.Ticker
	deploymentRiskTickerDuration time.Duration

	activeComponentStopped        concurrency.Signal
	activeComponentTicker         *time.Ticker
	activeComponentTickerDuration time.Duration
	acUpdater                     activeComponentsUpdater.Updater

	nodes        nodeDatastore.DataStore
	nodeEnricher nodeEnricher.NodeEnricher

	shortCircuitSig   concurrency.Signal
	stopSig           concurrency.Signal
	riskStopped       concurrency.Signal
	enrichmentStopped concurrency.Signal

	signatureVerificationSig concurrency.Signal
	// used for testing
	reprocessingStarted  concurrency.Signal
	reprocessingComplete concurrency.Signal

	connManager connection.Manager
	indexQueue  queue.WaitableQueue
}

func (l *loopImpl) ReprocessRiskForDeployments(deploymentIDs ...string) {
	l.deploymentRiskLock.Lock()
	defer l.deploymentRiskLock.Unlock()
	l.deploymentRiskSet.AddAll(deploymentIDs...)
}

// Start starts the enrich and detect loop.
func (l *loopImpl) Start() {
	l.enrichAndDetectTicker = time.NewTicker(l.enrichAndDetectTickerDuration)
	l.deploymentRiskTicker = time.NewTicker(l.deploymentRiskTickerDuration)

	go l.riskLoop()
	go l.enrichLoop()

	if features.ActiveVulnManagement.Enabled() {
		l.activeComponentTicker = time.NewTicker(l.activeComponentTickerDuration)
		go l.activeComponentLoop()
	}
}

// Stop stops the enrich and detect loop.
func (l *loopImpl) Stop() {
	l.stopSig.Signal()
	l.riskStopped.Wait()
	l.enrichmentStopped.Wait()
	if features.ActiveVulnManagement.Enabled() {
		l.activeComponentStopped.Wait()
	}
}

func (l *loopImpl) ShortCircuit() {
	// Signal that we should run a short circuited reprocessing. If the signal is already triggered, then the current
	// signal is effectively deduped
	l.shortCircuitSig.Signal()
}

func (l *loopImpl) ReprocessSignatureVerifications() {
	// Signal that we should reprocess signature verifications for all images. This will only trigger a reprocess with
	// refetch of signature verification results.
	// If the signal is already triggered, then the current signal is effectively deduped.
	if features.ImageSignatureVerification.Enabled() {
		l.signatureVerificationSig.Signal()
	}
}

func (l *loopImpl) sendDeployments(deploymentIDs []string) {
	query := search.NewQueryBuilder().AddStringsHighlighted(search.ClusterID, search.WildcardString)
	if len(deploymentIDs) > 0 {
		query = query.AddDocIDs(deploymentIDs...)
	}

	results, err := l.deployments.SearchDeployments(allAccessCtx, query.ProtoQuery())
	if err != nil {
		log.Errorf("error getting results for deployment reprocessing: %v", err)
		return
	}

	path, ok := deployments.OptionsMap.Get(search.ClusterID.String())
	if !ok {
		panic("No Cluster ID option for deployments")
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

		dedupeKey := uuid.NewV5(riskDedupeNamespace, r.Id).String()

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

func (l *loopImpl) runReprocessingForObjects(entityType string, getIDsFunc func() ([]string, error), individualReprocessFunc func(id string) bool) {
	if l.stopSig.IsDone() {
		return
	}
	ids, err := getIDsFunc()
	if err != nil {
		log.Errorf("Reprocessing failed: error retrieving active ids for %s: %v", entityType, err)
		return
	}
	log.Infof("Found %d %ss to scan", len(ids), entityType)

	sema := semaphore.NewWeighted(5)
	wg := concurrency.NewWaitGroup(0)
	nReprocessed := atomic.NewInt32(0)
	for _, id := range ids {
		wg.Add(1)
		if err := sema.Acquire(concurrency.AsContext(&l.stopSig), 1); err != nil {
			log.Errorf("context cancelled via stop: %v", err)
			return
		}
		go func(id string) {
			defer sema.Release(1)
			defer wg.Add(-1)
			if individualReprocessFunc(id) {
				nReprocessed.Inc()
				l.waitForIndexing()
			}
		}(id)
	}
	select {
	case <-wg.Done():
	case <-l.stopSig.Done():
		log.Info("Stopping reprocessing due to stop signal")
		return
	}

	log.Infof("Successfully reprocessed %d/%d %ss", nReprocessed.Load(), len(ids), entityType)
}

func (l *loopImpl) reprocessImage(id string, fetchOpt imageEnricher.FetchOption,
	reprocessingFunc imageReprocessingFunc) (*storage.Image, bool) {
	image, exists, err := l.images.GetImage(allAccessCtx, id)
	if err != nil {
		log.Errorf("error fetching image %q from the database: %v", id, err)
		return nil, false
	}
	if !exists || image.GetNotPullable() || image.GetIsClusterLocal() {
		return nil, false
	}

	result, err := reprocessingFunc(emptyCtx, imageEnricher.EnrichmentContext{
		FetchOpt: fetchOpt,
	}, image)

	if err != nil {
		log.Errorf("error enriching image: %v", err)
		return nil, false
	}
	if result.ImageUpdated {
		if err := l.risk.CalculateRiskAndUpsertImage(image); err != nil {
			log.Errorf("error upserting image %q into datastore: %v", image.GetName().GetFullName(), err)
			return nil, false
		}
	}

	l.waitForIndexing()

	return image, true
}

func (l *loopImpl) getActiveImageIDs() ([]string, error) {
	query := search.NewQueryBuilder().AddStringsHighlighted(search.DeploymentID, search.WildcardString).ProtoQuery()
	results, err := l.images.Search(allAccessCtx, query)
	if err != nil {
		return nil, errors.Wrap(err, "error searching for active image IDs")
	}

	return search.ResultsToIDs(results), nil
}

func (l *loopImpl) waitForIndexing() {
	indexingCompleted := concurrency.NewSignal()
	l.indexQueue.PushSignal(&indexingCompleted)

	select {
	case <-indexingCompleted.Done():
	case <-l.stopSig.Done():
	}
}

func (l *loopImpl) reprocessImagesAndResyncDeployments(fetchOpt imageEnricher.FetchOption,
	imgReprocessingFunc imageReprocessingFunc, imageQuery *v1.Query) {
	if l.stopSig.IsDone() {
		return
	}
	results, err := l.images.Search(allAccessCtx, imageQuery)
	if err != nil {
		log.Errorf("error searching for active image IDs: %v", err)
		return
	}

	log.Infof("Found %d images to scan", len(results))
	if len(results) == 0 {
		return
	}

	sema := semaphore.NewWeighted(5)
	wg := concurrency.NewWaitGroup(0)
	nReprocessed := atomic.NewInt32(0)
	for _, result := range results {
		wg.Add(1)
		if err := sema.Acquire(concurrency.AsContext(&l.stopSig), 1); err != nil {
			log.Errorf("context cancelled via stop: %v", err)
			return
		}
		// Duplicates can exist if the image is within multiple deployments
		clusterIDSet := set.NewStringSet(result.Matches[imageClusterIDFieldPath]...)
		go func(id string, clusterIDs set.StringSet) {
			defer sema.Release(1)
			defer wg.Add(-1)

			image, successfullyProcessed := l.reprocessImage(id, fetchOpt, imgReprocessingFunc)
			if !successfullyProcessed {
				return
			}
			nReprocessed.Inc()

			utils.FilterSuppressedCVEsNoClone(image)
			utils.StripCVEDescriptionsNoClone(image)

			for clusterID := range clusterIDs {
				conn := l.connManager.GetConnection(clusterID)
				if conn == nil {
					continue
				}
				err := conn.InjectMessage(concurrency.AsContext(&l.stopSig), &central.MsgToSensor{
					Msg: &central.MsgToSensor_UpdatedImage{
						UpdatedImage: image,
					},
				})
				if err != nil {
					log.Errorf("error injecting updated image %s to Sensor %q: %v", image.GetName().GetFullName(), clusterID, err)
				}
			}
		}(result.ID, clusterIDSet)
	}
	select {
	case <-wg.Done():
	case <-l.stopSig.Done():
		log.Info("Stopping reprocessing due to stop signal")
		return
	}
	log.Infof("Successfully reprocessed %d/%d images", nReprocessed.Load(), len(results))
	log.Infof("Resyncing deployments now that images have been reprocessed...")
	// Once the images have been rescanned, then reprocess the deployments.
	// This should not take a particularly long period of time.
	if !l.stopSig.IsDone() {
		l.connManager.BroadcastMessage(&central.MsgToSensor{
			Msg: &central.MsgToSensor_ReprocessDeployments{
				ReprocessDeployments: &central.ReprocessDeployments{},
			},
		})
	}
}

func (l *loopImpl) reprocessNode(id string) bool {
	node, exists, err := l.nodes.GetNode(allAccessCtx, id)
	if err != nil {
		log.Errorf("error fetching node %q from the database: %v", id, err)
		return false
	}
	if !exists {
		log.Warnf("node %q does not exist in the database. Skipping...", id)
		return false
	}

	err = l.nodeEnricher.EnrichNode(node)
	if err != nil {
		log.Errorf("error enriching node %s: %v", node.GetName(), err)
		return false
	}

	if err := l.risk.CalculateRiskAndUpsertNode(node); err != nil {
		log.Errorf("error upserting node %q into datastore: %v", node.GetName(), err)
		return false
	}

	return true
}

func (l *loopImpl) reprocessNodes() {
	l.runReprocessingForObjects("node", func() ([]string, error) {
		results, err := l.nodes.Search(allAccessCtx, search.EmptyQuery())
		if err != nil {
			return nil, err
		}
		return search.ResultsToIDs(results), nil
	}, l.reprocessNode)
}

func (l *loopImpl) reprocessWatchedImage(name string) bool {
	img, err := imageEnricher.EnrichImageByName(emptyCtx, l.imageEnricher, imageEnricher.EnrichmentContext{
		FetchOpt: imageEnricher.IgnoreExistingImages,
	}, name)
	if err != nil {
		log.Errorf("Error enriching watched image with name %q: %v", name, err)
		return false
	}
	// Save the image
	img.Id = utils.GetImageID(img)
	if img.GetId() == "" {
		return false
	}
	if err := l.risk.CalculateRiskAndUpsertImage(img); err != nil {
		log.Errorf("Failed to upsert watched image with name %q after enriching: %v", name, err)
		return false
	}
	return true
}

func (l *loopImpl) reprocessWatchedImages() {
	l.runReprocessingForObjects("watched image", func() ([]string, error) {
		watchedImages, err := l.watchedImages.GetAllWatchedImages(allAccessCtx)
		if err != nil {
			return nil, err
		}
		imageNames := make([]string, 0, len(watchedImages))
		for _, img := range watchedImages {
			imageNames = append(imageNames, img.GetName())
		}
		return imageNames, nil
	}, l.reprocessWatchedImage)
}

func (l *loopImpl) runReprocessing(imageFetchOpt imageEnricher.FetchOption) {
	l.reprocessingComplete.Reset()
	l.reprocessingStarted.Signal()

	l.reprocessNodes()
	l.reprocessWatchedImages()
	l.reprocessImagesAndResyncDeployments(imageFetchOpt, l.enrichImage, allImagesQuery)

	l.reprocessingStarted.Reset()
	l.reprocessingComplete.Signal()
}

func (l *loopImpl) runSignatureVerificationReprocessing() {
	l.reprocessWatchedImages()
	l.reprocessImagesAndResyncDeployments(imageEnricher.ForceRefetchSignaturesOnly,
		l.forceEnrichImageSignatureVerificationResults, imagesWithSignatureVerificationResultsQuery)

}

func (l *loopImpl) forceEnrichImageSignatureVerificationResults(ctx context.Context, _ imageEnricher.EnrichmentContext,
	image *storage.Image) (imageEnricher.EnrichmentResult, error) {
	return l.imageEnricher.EnrichWithSignatureVerificationData(ctx, image)
}

func (l *loopImpl) enrichImage(ctx context.Context, enrichCtx imageEnricher.EnrichmentContext,
	image *storage.Image) (imageEnricher.EnrichmentResult, error) {
	return l.imageEnricher.EnrichImage(ctx, enrichCtx, image)
}

func (l *loopImpl) enrichLoop() {
	defer l.enrichAndDetectTicker.Stop()
	defer l.enrichmentStopped.Signal()

	// Call runReprocessing with ForceRefetch on start to ensure that the image metadata reflects any changes
	// in the proto and to ensure that the images and nodes are pulling new scans on <= the reprocessing interval
	l.runReprocessing(imageEnricher.ForceRefetch)
	for !l.stopSig.IsDone() {
		select {
		case <-l.stopSig.Done():
			return
		case <-l.shortCircuitSig.Done():
			l.shortCircuitSig.Reset()
			l.runReprocessing(imageEnricher.UseCachesIfPossible)
		case <-l.signatureVerificationSig.Done():
			l.signatureVerificationSig.Reset()
			l.runSignatureVerificationReprocessing()
		case <-l.enrichAndDetectTicker.C:
			l.runReprocessing(imageEnricher.ForceRefetchCachedValuesOnly)
		}
	}
}

func (l *loopImpl) riskLoop() {
	defer l.riskStopped.Signal()
	defer l.deploymentRiskTicker.Stop()

	for !l.stopSig.IsDone() {
		select {
		case <-l.stopSig.Done():
			return
		case <-l.deploymentRiskTicker.C:
			l.deploymentRiskLock.Lock()
			if l.deploymentRiskSet.Cardinality() > 0 {
				// goroutine to ensure this is non-blocking.
				go l.sendDeployments(l.deploymentRiskSet.AsSlice())
				l.deploymentRiskSet.Clear()
			}
			l.deploymentRiskLock.Unlock()
		}
	}
}

func (l *loopImpl) activeComponentLoop() {
	defer l.activeComponentStopped.Signal()
	defer l.activeComponentTicker.Stop()

	for !l.stopSig.IsDone() {
		select {
		case <-l.stopSig.Done():
			return
		case <-l.activeComponentTicker.C:
			l.acUpdater.Update()
		}
	}
}
