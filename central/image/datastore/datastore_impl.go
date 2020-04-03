package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/central/image/index"
	pkgImgComponent "github.com/stackrox/rox/central/imagecomponent"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

const (
	imageBatchSize = 500
)

var (
	log = logging.LoggerForModule()

	imagesSAC = sac.ForResource(resources.Image)
)

type datastoreImpl struct {
	keyedMutex *concurrency.KeyedMutex

	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher

	components imageComponentDS.DataStore
	risks      riskDS.DataStore

	imageRanker          *ranking.Ranker
	imageComponentRanker *ranking.Ranker
}

func newDatastoreImpl(storage store.Store, indexer index.Indexer, searcher search.Searcher,
	imageComponents imageComponentDS.DataStore, risks riskDS.DataStore,
	imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) (*datastoreImpl, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,

		components: imageComponents,
		risks:      risks,

		imageRanker:          imageRanker,
		imageComponentRanker: imageComponentRanker,

		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "Search")

	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "SearchImages")

	return ds.searcher.SearchImages(ctx, q)
}

// SearchRawImages delegates to the underlying searcher.
func (ds *datastoreImpl) SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "SearchRawImages")

	imgs, err := ds.searcher.SearchRawImages(ctx, q)
	if err != nil {
		return nil, err
	}

	ds.updateImagePriority(imgs...)

	return imgs, nil
}

func (ds *datastoreImpl) SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "SearchListImages")

	imgs, err := ds.searcher.SearchListImages(ctx, q)
	if err != nil {
		return nil, err
	}

	ds.updateListImagePriority(imgs...)

	return imgs, nil
}

func (ds *datastoreImpl) ListImage(ctx context.Context, sha string) (*storage.ListImage, bool, error) {
	img, found, err := ds.storage.ListImage(sha)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := ds.canReadImage(ctx, sha); err != nil || !ok {
		return nil, false, err
	}

	ds.updateListImagePriority(img)

	return img, true, nil
}

// CountImages delegates to the underlying store.
func (ds *datastoreImpl) CountImages(ctx context.Context) (int, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return ds.storage.CountImages()
	}

	searchResults, err := ds.Search(ctx, pkgSearch.EmptyQuery())
	if err != nil {
		return 0, err
	}

	return len(searchResults), nil
}

func (ds *datastoreImpl) canReadImage(ctx context.Context, sha string) (bool, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	queryForImage := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, sha).ProtoQuery()
	if results, err := ds.searcher.Search(ctx, queryForImage); err != nil {
		return false, err
	} else if len(results) > 0 {
		return true, nil
	}

	return false, nil
}

// GetImage delegates to the underlying store.
func (ds *datastoreImpl) GetImage(ctx context.Context, sha string) (*storage.Image, bool, error) {
	img, found, err := ds.storage.GetImage(sha)
	if err != nil || !found {
		return nil, false, err
	}
	if ok, err := ds.canReadImage(ctx, sha); err != nil || !ok {
		return nil, false, err
	}

	ds.updateImagePriority(img)

	return img, true, nil
}

// GetImagesBatch delegates to the underlying store.
func (ds *datastoreImpl) GetImagesBatch(ctx context.Context, shas []string) ([]*storage.Image, error) {
	var imgs []*storage.Image
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		imgs, _, err = ds.storage.GetImagesBatch(shas)
		if err != nil {
			return nil, err
		}
	} else {
		shasQuery := pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ImageSHA, shas...).ProtoQuery()
		imgs, err = ds.SearchRawImages(ctx, shasQuery)
		if err != nil {
			return nil, err
		}
	}

	ds.updateImagePriority(imgs...)

	return imgs, nil
}

// UpsertImage dedupes the image with the underlying storage and adds the image to the index.
func (ds *datastoreImpl) UpsertImage(ctx context.Context, image *storage.Image) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "UpsertImage")

	if image.GetId() == "" {
		return errors.New("cannot upsert an image without an id")
	}

	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.keyedMutex.Lock(image.GetId())
	defer ds.keyedMutex.Unlock(image.GetId())

	oldImage, exists, err := ds.storage.GetImage(image.GetId())
	if err != nil {
		return err
	}

	// Update image with latest risk score
	image.RiskScore = ds.imageRanker.GetScoreForID(image.GetId())
	// If the merge causes no changes, then no reason to save
	if exists && !merge(image, oldImage) {
		return nil
	}

	ds.updateComponentRisk(image)
	enricher.FillScanStats(image)

	if err = ds.storage.Upsert(image); err != nil {
		return err
	}
	if features.Dackbox.Enabled() {
		return nil
	}
	if err := ds.indexer.AddImage(image); err != nil {
		return err
	}
	return ds.storage.AckKeysIndexed(image.GetId())
}

func (ds *datastoreImpl) DeleteImages(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "DeleteImages")

	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	errorList := errorhelpers.NewErrorList("deleting images")
	deleteRiskCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resources.Risk)))

	for _, id := range ids {
		if err := ds.storage.Delete(id); err != nil {
			errorList.AddError(err)
			continue
		}
		if !features.Dackbox.Enabled() {
			errorList.AddError(ds.indexer.DeleteImage(id))
			errorList.AddError(ds.storage.AckKeysIndexed(id))
		} else {
			if err := ds.risks.RemoveRisk(deleteRiskCtx, id, storage.RiskSubjectType_IMAGE); err != nil {
				return err
			}
		}
	}
	// removing component risk handled by pruning
	return errorList.ToError()
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "Exists")

	if ok, err := ds.canReadImage(ctx, id); err != nil || !ok {
		return false, err
	}
	return ds.storage.Exists(id)
}

// merge adds the most up to date data from the two inputs to the first input.
func merge(mergeTo *storage.Image, mergeWith *storage.Image) (updated bool) {
	// If the image currently in the DB has more up to date info, swap it out.
	if mergeWith.GetRiskScore() != mergeTo.GetRiskScore() {
		updated = true
	}
	if mergeWith.GetMetadata().GetV1().GetCreated().Compare(mergeTo.GetMetadata().GetV1().GetCreated()) > 0 {
		mergeTo.Metadata = mergeWith.GetMetadata()
	} else {
		updated = true
	}
	if mergeWith.GetScan().GetScanTime().Compare(mergeTo.GetScan().GetScanTime()) > 0 {
		mergeTo.Scan = mergeWith.GetScan()
	} else {
		updated = true
	}

	return
}

func (ds *datastoreImpl) fullReindex() error {
	log.Info("[STARTUP] Reindexing all images")

	images, err := ds.storage.GetImages()
	if err != nil {
		return err
	}
	log.Infof("[STARTUP] Found %d images to index", len(images))
	imageBatcher := batcher.New(len(images), imageBatchSize)
	for start, end, valid := imageBatcher.Next(); valid; start, end, valid = imageBatcher.Next() {
		if err := ds.indexer.AddImages(images[start:end]); err != nil {
			return err
		}
		log.Infof("[STARTUP] Successfully indexed %d/%d images", end, len(images))
	}
	log.Infof("[STARTUP] Successfully indexed %d images", len(images))

	// Clear the keys because we just re-indexed everything
	keys, err := ds.storage.GetKeysToIndex()
	if err != nil {
		return err
	}
	if err := ds.storage.AckKeysIndexed(keys...); err != nil {
		return err
	}

	// Write out that initial indexing is complete
	if err := ds.indexer.MarkInitialIndexingComplete(); err != nil {
		return err
	}

	return nil
}

func (ds *datastoreImpl) buildIndex() error {
	if features.Dackbox.Enabled() {
		return nil
	}
	defer debug.FreeOSMemory()
	needsFullIndexing, err := ds.indexer.NeedsInitialIndexing()
	if err != nil {
		return err
	}
	if needsFullIndexing {
		return ds.fullReindex()
	}

	log.Info("[STARTUP] Determining if image db/indexer reconciliation is needed")

	imagesToIndex, err := ds.storage.GetKeysToIndex()
	if err != nil {
		return errors.Wrap(err, "error retrieving keys to index")
	}

	log.Infof("[STARTUP] Found %d Images to index", len(imagesToIndex))

	imageBatcher := batcher.New(len(imagesToIndex), imageBatchSize)
	for start, end, valid := imageBatcher.Next(); valid; start, end, valid = imageBatcher.Next() {
		images, missingIndices, err := ds.storage.GetImagesBatch(imagesToIndex[start:end])
		if err != nil {
			return err
		}
		if err := ds.indexer.AddImages(images); err != nil {
			return err
		}

		if len(missingIndices) > 0 {
			idsToRemove := make([]string, 0, len(missingIndices))
			for _, missingIdx := range missingIndices {
				idsToRemove = append(idsToRemove, imagesToIndex[start:end][missingIdx])
			}
			if err := ds.indexer.DeleteImages(idsToRemove); err != nil {
				return err
			}
		}

		// Ack keys so that even if central restarts, we don't need to reindex them again
		if err := ds.storage.AckKeysIndexed(imagesToIndex[start:end]...); err != nil {
			return err
		}
		log.Infof("[STARTUP] Successfully indexed %d/%d images", end, len(imagesToIndex))
	}

	log.Info("[STARTUP] Successfully indexed all out of sync images")
	return nil
}

func (ds *datastoreImpl) initializeRankers() {
	readCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Image)))

	results, err := ds.searcher.Search(readCtx, pkgSearch.EmptyQuery())
	if err != nil {
		log.Error(err)
		return
	}

	for _, id := range pkgSearch.ResultsToIDs(results) {
		image, found, err := ds.storage.GetImage(id)
		if err != nil {
			log.Error(err)
			continue
		} else if !found {
			continue
		}

		ds.imageRanker.Add(id, image.GetRiskScore())
	}
}

func (ds *datastoreImpl) updateListImagePriority(images ...*storage.ListImage) {
	for _, image := range images {
		image.Priority = ds.imageRanker.GetRankForID(image.GetId())
	}
}

func (ds *datastoreImpl) updateImagePriority(images ...*storage.Image) {
	for _, image := range images {
		image.Priority = ds.imageRanker.GetRankForID(image.GetId())
	}
}

func (ds *datastoreImpl) updateComponentRisk(image *storage.Image) {
	if !features.Dackbox.Enabled() {
		return
	}
	for _, component := range image.GetScan().GetComponents() {
		component.RiskScore = ds.imageComponentRanker.GetScoreForID(pkgImgComponent.ComponentID{
			Name:    component.GetName(),
			Version: component.GetVersion(),
		}.ToString())
	}
}
