package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
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

	risks riskDS.DataStore

	imageRanker          *ranking.Ranker
	imageComponentRanker *ranking.Ranker
}

func newDatastoreImpl(storage store.Store, indexer index.Indexer, searcher search.Searcher, risks riskDS.DataStore,
	imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) *datastoreImpl {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,

		risks: risks,

		imageRanker:          imageRanker,
		imageComponentRanker: imageComponentRanker,

		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	return ds
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "Search")

	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "Count")

	return ds.searcher.Count(ctx, q)
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

	return ds.Count(ctx, pkgSearch.EmptyQuery())
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

// GetImageMetadata gets the image data without the scan
func (ds *datastoreImpl) GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
	img, found, err := ds.storage.GetImageMetadata(id)
	if err != nil || !found {
		return nil, false, err
	}
	if ok, err := ds.canReadImage(ctx, id); err != nil || !ok {
		return nil, false, err
	}
	ds.updateImagePriority(img)

	return img, true, nil
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
		return sac.ErrResourceAccessDenied
	}

	ds.keyedMutex.Lock(image.GetId())
	defer ds.keyedMutex.Unlock(image.GetId())

	ds.updateComponentRisk(image)
	enricher.FillScanStats(image)

	if err := ds.storage.Upsert(image); err != nil {
		return err
	}
	// If the image in db is latest, this image object will be carrying its risk score
	ds.imageRanker.Add(image.GetId(), image.GetRiskScore())
	return nil
}

func (ds *datastoreImpl) DeleteImages(ctx context.Context, ids ...string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), "Image", "DeleteImages")

	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	errorList := errorhelpers.NewErrorList("deleting images")
	deleteRiskCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resources.Risk)))

	for _, id := range ids {
		if err := ds.storage.Delete(id); err != nil {
			errorList.AddError(err)
			continue
		}
		if err := ds.risks.RemoveRisk(deleteRiskCtx, id, storage.RiskSubjectType_IMAGE); err != nil {
			return err
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
		image, found, err := ds.storage.GetImageMetadata(id)
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
	for _, component := range image.GetScan().GetComponents() {
		component.RiskScore = ds.imageComponentRanker.GetScoreForID(scancomponent.ComponentID(component.GetName(), component.GetVersion()))
	}
}
