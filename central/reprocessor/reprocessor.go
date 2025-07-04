package reprocessor

import (
	"context"
	"time"

	"github.com/pkg/errors"
	activeComponentsUpdater "github.com/stackrox/rox/central/activecomponent/updater"
	administrationEvents "github.com/stackrox/rox/central/administration/events"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/metrics"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/maputil"
	nodeEnricher "github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	imageMapping "github.com/stackrox/rox/pkg/search/options/images"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/atomic"
	"golang.org/x/sync/semaphore"
)

const (
	imageReprocessorSemaphoreSize = int64(5)
)

var (
	log = logging.LoggerForModule(administrationEvents.EnableAdministrationEvents())

	riskDedupeNamespace = uuid.NewV4()

	once sync.Once
	loop Loop

	allAccessCtx = sac.WithAllAccess(context.Background())

	emptyCtx = context.Background()

	delegateScanCtx = sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
		),
	)

	imageClusterIDFieldPath = imageMapping.ImageDeploymentOptions.MustGet(search.ClusterID.String()).GetFieldPath()

	allImagesQuery = search.NewQueryBuilder().AddStringsHighlighted(search.ClusterID, search.WildcardString).
			ProtoQuery()

	imagesWithSignaturesQuery = search.NewQueryBuilder().
		// We take all images into account irrespective whether they have a cluster associated with them
		// or not. The reason is that we want to reprocess those in case e.g. a previous signature
		// verification failure lead to an enforcement, which would make the image not have any cluster
		// associated with it.
		AddTimeRangeField(search.ImageSignatureFetchedTime,
			// Could potentially miss images that _just_ fetched signatures so creating a small jitter
			// to include those as well.
			time.Unix(0, 0), time.Now().Add(10*time.Second)).
		ProtoQuery()
)

// Singleton returns the singleton reprocessor loop
func Singleton() Loop {
	once.Do(func() {
		loop = NewLoop(connection.ManagerSingleton(), enrichment.ImageEnricherSingleton(), enrichment.NodeEnricherSingleton(),
			deploymentDatastore.Singleton(), imageDatastore.Singleton(), nodeDatastore.Singleton(), manager.Singleton(),
			watchedImageDataStore.Singleton(), activeComponentsUpdater.Singleton())
	})
	return loop
}

// Loop combines periodically (every 4 hours by default) runs enrichment and detection.
//
//go:generate mockgen-wrapper
type Loop interface {
	Start()
	ShortCircuit()
	Stop()

	ReprocessRiskForDeployments(deploymentIDs ...string)
	ReprocessSignatureVerifications(firstIntegration bool)
}

// NewLoop returns a new instance of a Loop.
func NewLoop(connManager connection.Manager, imageEnricher imageEnricher.ImageEnricher, nodeEnricher nodeEnricher.NodeEnricher,
	deployments deploymentDatastore.DataStore, images imageDatastore.DataStore, nodes nodeDatastore.DataStore,
	risk manager.Manager, watchedImages watchedImageDataStore.DataStore, acUpdater activeComponentsUpdater.Updater) Loop {
	return newLoopWithDuration(
		connManager, imageEnricher, nodeEnricher, deployments, images, nodes, risk,
		watchedImages, env.ReprocessInterval.DurationSetting(), env.RiskReprocessInterval.DurationSetting(),
		env.ActiveVulnRefreshInterval.DurationSetting(), acUpdater)
}

// newLoopWithDuration returns a loop that ticks at the given duration.
// It is NOT exported, since we don't want clients to control the duration; it only exists as a separate function
// to enable testing.
func newLoopWithDuration(connManager connection.Manager, imageEnricher imageEnricher.ImageEnricher, nodeEnricher nodeEnricher.NodeEnricher,
	deployments deploymentDatastore.DataStore, images imageDatastore.DataStore, nodes nodeDatastore.DataStore,
	risk manager.Manager, watchedImages watchedImageDataStore.DataStore, enrichAndDetectDuration, deploymentRiskDuration,
	activeComponentTickerDuration time.Duration, acUpdater activeComponentsUpdater.Updater) *loopImpl {
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

		connManager: connManager,

		injectMessageTimeoutDur: env.ReprocessInjectMessageTimeout.DurationSetting(),
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

	signatureVerificationSig  concurrency.Signal
	firstSignatureIntegration concurrency.Flag

	reprocessingInProgress concurrency.Flag

	connManager connection.Manager

	injectMessageTimeoutDur time.Duration
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

	l.activeComponentTicker = time.NewTicker(l.activeComponentTickerDuration)
	go l.activeComponentLoop()
}

// Stop stops the enrich and detect loop.
func (l *loopImpl) Stop() {
	l.stopSig.Signal()
	l.riskStopped.Wait()
	l.enrichmentStopped.Wait()
	l.activeComponentStopped.Wait()
}

func (l *loopImpl) ShortCircuit() {
	// Signal that we should run a short circuited reprocessing. If the signal is already triggered, then the current
	// signal is effectively deduped
	l.shortCircuitSig.Signal()
}

func (l *loopImpl) ReprocessSignatureVerifications(firstIntegration bool) {
	// Signal that we should reprocess signature verifications for all images. This will only trigger a reprocess with
	// refetch of signature verification results.
	// If the signal is already triggered, then the current signal is effectively deduped.
	l.firstSignatureIntegration.Set(firstIntegration)
	l.signatureVerificationSig.Signal()
}

func (l *loopImpl) sendDeployments(deploymentIDs []string) {
	query := search.NewQueryBuilder().AddStringsHighlighted(search.ClusterID, search.WildcardString)
	if len(deploymentIDs) > 0 {
		query = query.AddDocIDs(deploymentIDs...)
	}

	results, err := l.deployments.SearchDeployments(allAccessCtx, query.ProtoQuery())
	if err != nil {
		log.Errorw("Error getting results for deployment reprocessing", logging.Err(err))
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
		log.Errorw("Failed to retrieve active IDs for entity", logging.String("entity", entityType),
			logging.Err(err))
		return
	}
	log.Infof("Found %d %ss to scan", len(ids), entityType)

	sema := semaphore.NewWeighted(5)
	wg := concurrency.NewWaitGroup(0)
	nReprocessed := atomic.NewInt32(0)
	for _, id := range ids {
		wg.Add(1)
		if err := sema.Acquire(concurrency.AsContext(&l.stopSig), 1); err != nil {
			log.Errorw("Reprocessing stopped", logging.Err(err))
			return
		}
		go func(id string) {
			defer sema.Release(1)
			defer wg.Add(-1)
			if individualReprocessFunc(id) {
				nReprocessed.Inc()
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
		log.Errorw("Error fetching image from database", logging.ImageID(id), logging.Err(err))
		return nil, false
	}
	if !exists || image.GetNotPullable() || image.GetIsClusterLocal() {
		return nil, false
	}

	result, err := reprocessingFunc(emptyCtx, imageEnricher.EnrichmentContext{
		FetchOpt: fetchOpt,
	}, image)

	if err != nil {
		log.Errorw("Error enriching image", logging.ImageName(image.GetName().GetFullName()), logging.ImageID(image.GetId()), logging.Err(err))
		return nil, false
	}
	if result.ImageUpdated {
		if err := l.risk.CalculateRiskAndUpsertImage(image); err != nil {
			log.Errorw("Error upserting image into datastore",
				logging.ImageName(image.GetName().GetFullName()), logging.ImageID(image.GetId()), logging.Err(err))
			return nil, false
		}
		// We need to fetch the image again to make sure all fields are populated.
		// GetImage will internally call a Merge function which will use the CVEEdges table to enrich fields like
		// FirstImageOccurrence and FirstSystemOccurrence.
		newImage, exists, err := l.images.GetImage(allAccessCtx, id)
		if err != nil {
			log.Errorw("Error fetching image from database", logging.ImageName(image.GetName().GetFullName()), logging.ImageID(image.GetId()), logging.Err(err))
			return nil, false
		}
		if !exists {
			log.Errorw("The image was not found after enrichement", logging.ImageName(image.GetName().GetFullName()), logging.ImageID(image.GetId()))
			return nil, false
		}
		return newImage, true
	}
	return image, true
}

func (l *loopImpl) reprocessImagesAndResyncDeployments(fetchOpt imageEnricher.FetchOption,
	imgReprocessingFunc imageReprocessingFunc, imageQuery *v1.Query) {
	if l.stopSig.IsDone() {
		return
	}
	results, err := l.images.Search(allAccessCtx, imageQuery)
	if err != nil {
		log.Errorw("Error searching for active image IDs", logging.Err(err))
		return
	}

	log.Infof("Found %d images to scan", len(results))
	if len(results) == 0 {
		return
	}

	sema := semaphore.NewWeighted(imageReprocessorSemaphoreSize)
	wg := concurrency.NewWaitGroup(0)
	nReprocessed := atomic.NewInt32(0)
	skipClusterIDs := maputil.NewSyncMap[string, struct{}]()
	for _, result := range results {
		wg.Add(1)
		if err := sema.Acquire(concurrency.AsContext(&l.stopSig), 1); err != nil {
			log.Errorw("Reprocessing stopped", logging.Err(err))
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

			// Send the updated image to relevant clusters.
			for clusterID := range clusterIDs {
				conn := l.connManager.GetConnection(clusterID)
				if conn == nil {
					continue
				}

				msg := &central.MsgToSensor{
					Msg: &central.MsgToSensor_UpdatedImage{
						UpdatedImage: image,
					},
				}

				// If were prior errors, do not attempt to send a message to this cluster.
				if skipClusterIDs.Contains(clusterID) {
					metrics.IncrementMsgToSensorNotSentCounter(clusterID, msg, metrics.NotSentSkip)
					log.Debugw("Not sending updated image to cluster due to prior errors",
						logging.ImageID(image.GetId()),
						logging.ImageName(image.GetName().GetFullName()),
						logging.String("dst_cluster", clusterID),
					)
					continue
				}

				err := l.injectMessage(concurrency.AsContext(&l.stopSig), conn, msg)
				if err != nil {
					skipClusterIDs.Store(clusterID, struct{}{})
					log.Errorw("Error sending updated image to cluster, skipping cluster until next reprocessing cycle",
						logging.ImageName(image.GetName().GetFullName()),
						logging.ImageID(image.GetId()), logging.Err(err),
						// Not using logging.ClusterID() to avoid "duplicate resource ID field found" panic
						logging.String("dst_cluster", clusterID),
					)
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
	log.Info("Resyncing deployments now that images have been reprocessed...")
	// Once the images have been rescanned, then reprocess the deployments.
	// This should not take a particularly long period of time.
	if !l.stopSig.IsDone() {
		msg := &central.MsgToSensor{
			Msg: &central.MsgToSensor_ReprocessDeployments{
				ReprocessDeployments: &central.ReprocessDeployments{},
			},
		}
		ctx := concurrency.AsContext(&l.stopSig)
		for _, conn := range l.connManager.GetActiveConnections() {
			clusterID := conn.ClusterID()
			if skipClusterIDs.Contains(clusterID) {
				metrics.IncrementMsgToSensorNotSentCounter(clusterID, msg, metrics.NotSentSkip)
				log.Errorw("Not sending reprocess deployments to cluster due to prior errors",
					logging.ClusterID(clusterID),
				)
				continue
			}

			err := l.injectMessage(ctx, conn, msg)
			if err != nil {
				log.Errorw("Error sending reprocess deployments message to cluster",
					logging.ClusterID(clusterID),
					logging.Err(err),
				)
			}
		}
	}
}

// injectMessage will inject a message onto connection, an error will be returned if the
// injection fails for any reason, including timeout.
func (l *loopImpl) injectMessage(ctx context.Context, conn connection.SensorConnection, msg *central.MsgToSensor) error {
	if l.injectMessageTimeoutDur > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, l.injectMessageTimeoutDur)
		defer cancel()
	}

	err := conn.InjectMessage(ctx, msg)
	if err != nil {
		return errors.Wrap(err, "injecting message to sensor")
	}

	return nil
}

func (l *loopImpl) reprocessNode(id string) bool {
	node, exists, err := l.nodes.GetNode(allAccessCtx, id)
	if err != nil {
		log.Errorw("Error fetching node from the database", logging.NodeID(id), logging.Err(err))
		return false
	}
	if !exists {
		log.Warnw("Error fetching non-existing node from the database", logging.NodeID(id))
		return false
	}

	if nodeEnricher.SupportsNodeScanning(node) {
		log.Infof("node %s is host-scanned: skipping reprocess", nodeDatastore.NodeString(node))
		// False signals there was no writes to the database and no actual reprocessing.
		return false
	}

	err = l.nodeEnricher.EnrichNode(node)
	if err != nil {
		log.Errorw("Error enriching node", logging.String("node", nodeDatastore.NodeString(node)), logging.Err(err))
		return false
	}
	if err := l.risk.CalculateRiskAndUpsertNode(node); err != nil {
		log.Error(err)
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
	enrichmentCtx := imageEnricher.EnrichmentContext{
		FetchOpt: imageEnricher.IgnoreExistingImages,
	}

	ctx := emptyCtx
	if features.DelegateWatchedImageReprocessing.Enabled() {
		ctx = delegateScanCtx
		enrichmentCtx.Delegable = true
	}

	img, err := imageEnricher.EnrichImageByName(ctx, l.imageEnricher, enrichmentCtx, name)
	if err != nil {
		log.Errorw("Error enriching watched image", logging.ImageName(name), logging.Err(err))
		return false
	}
	// Save the image
	img.Id = utils.GetSHA(img)
	if img.GetId() == "" {
		return false
	}
	if err := l.risk.CalculateRiskAndUpsertImage(img); err != nil {
		log.Errorw("Error upserting watched image after enriching", logging.ImageName(name), logging.ImageID(img.GetId()), logging.Err(err))
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
	// In case the current reprocessing run takes longer than the ticker (i.e. > 4 hours when using a high number of
	// images), we shouldn't trigger a parallel reprocessing run.
	if l.reprocessingInProgress.TestAndSet(true) {
		return
	}
	defer metrics.SetReprocessorDuration(time.Now())
	l.reprocessNodes()
	l.reprocessWatchedImages()
	l.reprocessImagesAndResyncDeployments(imageFetchOpt, l.enrichImage, allImagesQuery)

	l.reprocessingInProgress.Set(false)
}

func (l *loopImpl) runSignatureVerificationReprocessing() {
	defer metrics.SetSignatureVerificationReprocessorDuration(time.Now())
	l.reprocessWatchedImages()
	query := imagesWithSignaturesQuery
	// If we have reprocessed when the _first_ signature integration is added, then take into account all images.
	if l.firstSignatureIntegration.Get() {
		query = allImagesQuery
	}

	l.reprocessImagesAndResyncDeployments(imageEnricher.ForceRefetchSignaturesOnly,
		l.forceEnrichImageSignatureVerificationResults, query)

	l.firstSignatureIntegration.Set(false)
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
			concurrency.WithLock(&l.deploymentRiskLock, func() {
				if l.deploymentRiskSet.Cardinality() > 0 {
					// goroutine to ensure this is non-blocking.
					go l.sendDeployments(l.deploymentRiskSet.AsSlice())
					l.deploymentRiskSet.Clear()
				}
			})
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
