package pruning

import (
	"context"
	"time"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

const (
	pruneImagesCheckInterval = 4 * time.Hour
	pruneImagesAfterDays     = 7
)

var (
	log = logging.LoggerForModule()
)

// GarbageCollector implements a generic garbage collection mechanism
type GarbageCollector interface {
	Start()
	Stop()
}

func newGarbageCollector(images imageDatastore.DataStore, deployments deploymentDatastore.DataStore) GarbageCollector {
	return &garbageCollectorImpl{
		images:      images,
		deployments: deployments,

		stopSig:    concurrency.NewSignal(),
		stoppedSig: concurrency.NewSignal(),
	}
}

type garbageCollectorImpl struct {
	images      imageDatastore.DataStore
	deployments deploymentDatastore.DataStore

	stopSig    concurrency.Signal
	stoppedSig concurrency.Signal
}

func (g *garbageCollectorImpl) Start() {
	go g.runImageGC()
}

func (g *garbageCollectorImpl) runImageGC() {
	// Run collection initially then run on a ticker
	g.collectImages()
	t := time.NewTicker(pruneImagesCheckInterval)
	for {
		select {
		case <-t.C:
			g.collectImages()
		case <-g.stopSig.Done():
			g.stoppedSig.Done()
			return
		}
	}
}

func (g *garbageCollectorImpl) collectImages() {
	qb := search.NewQueryBuilder().AddDays(search.LastUpdatedTime, pruneImagesAfterDays).ProtoQuery()
	imageResults, err := g.images.Search(context.TODO(), qb)
	if err != nil {
		log.Error(err)
		return
	}

	var imagesToPrune []string
	for _, result := range imageResults {
		q := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, result.ID).ProtoQuery()
		results, err := g.deployments.Search(context.TODO(), q)
		if err != nil {
			log.Error(err)
			continue
		}
		// If there are no deployment queries that match, then allow the image to be pruned
		if len(results) == 0 {
			imagesToPrune = append(imagesToPrune, result.ID)
		}
	}
	if len(imagesToPrune) > 0 {
		log.Infof("Image Pruner will be removing the following images: %+v", imagesToPrune)
		if err := g.images.DeleteImages(context.TODO(), imagesToPrune...); err != nil {
			log.Error(err)
		}
	}
}

func (g *garbageCollectorImpl) Stop() {
	g.stopSig.Signal()
	<-g.stoppedSig.Done()
}
