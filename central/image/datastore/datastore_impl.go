package datastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/txn"
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

	componentsInImages map[string]int
}

func newDatastoreImpl(storage store.Store, indexer index.Indexer, searcher search.Searcher, risks riskDS.DataStore) (*datastoreImpl, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
		risks:    risks,

		componentsInImages: make(map[string]int),
		keyedMutex:         concurrency.NewKeyedMutex(16),
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

func (ds *datastoreImpl) initializeRankers() error {
	ds.imageRanker = ranking.ImageRanker()
	ds.imageComponentRanker = ranking.ImageComponentRanker()

	riskElevatedCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Risk),
		))

	imageRisks, err := ds.risks.SearchRawRisks(riskElevatedCtx, searchPkg.NewQueryBuilder().AddStrings(
		searchPkg.RiskSubjectType, storage.RiskSubjectType_IMAGE.String()).ProtoQuery())
	if err != nil {
		return err
	}
	for _, risk := range imageRisks {
		ds.imageRanker.Add(risk.GetSubject().GetId(), risk.GetScore())
	}

	imageComponentRisks, err := ds.risks.SearchRawRisks(riskElevatedCtx, searchPkg.NewQueryBuilder().AddStrings(
		searchPkg.RiskSubjectType, storage.RiskSubjectType_IMAGE_COMPONENT.String()).ProtoQuery())
	if err != nil {
		return err
	}
	for _, risk := range imageComponentRisks {
		ds.imageComponentRanker.Add(risk.GetSubject().GetId(), risk.GetScore())
	}
	return nil
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchImages(ctx, q)
}

// SearchRawImages delegates to the underlying searcher.
func (ds *datastoreImpl) SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error) {
	imgs, err := ds.searcher.SearchRawImages(ctx, q)
	if err != nil {
		return nil, err
	}
	ds.updateImagePriority(imgs...)
	return imgs, nil
}

func (ds *datastoreImpl) SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error) {
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

	searchResults, err := ds.Search(ctx, searchPkg.EmptyQuery())
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

	queryForImage := searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.ImageSHA, sha).ProtoQuery()
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
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		imgs, err := ds.storage.GetImagesBatch(shas)
		if err != nil {
			return nil, err
		}
		return imgs, nil
	}

	shasQuery := searchPkg.NewQueryBuilder().AddStrings(searchPkg.ImageSHA, shas...).ProtoQuery()
	imgs, err := ds.SearchRawImages(ctx, shasQuery)
	if err != nil {
		return nil, err
	}

	ds.updateImagePriority(imgs...)
	return imgs, nil
}

// UpsertImage dedupes the image with the underlying storage and adds the image to the index.
func (ds *datastoreImpl) UpsertImage(ctx context.Context, image *storage.Image) error {
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
	// If the merge causes no changes, then no reason to save
	if exists && !merge(image, oldImage) {
		return nil
	}

	enricher.FillScanStats(image)
	if err = ds.storage.UpsertImage(image); err != nil {
		return err
	}

	for _, imageComponent := range image.GetScan().GetComponents() {
		key := getImageComponentKey(imageComponent)
		ds.componentsInImages[key]++
	}

	return ds.indexer.AddImage(image)
}

func (ds *datastoreImpl) DeleteImages(ctx context.Context, ids ...string) error {
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	errorList := errorhelpers.NewErrorList("deleting images")
	deleteRiskCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Risk),
		))

	for _, id := range ids {
		image, found, err := ds.storage.GetImage(id)
		if err != nil || !found {
			return err
		}

		if err := ds.storage.DeleteImage(id); err != nil {
			errorList.AddError(err)
			continue
		}
		if err := ds.indexer.DeleteImage(id); err != nil {
			errorList.AddError(err)
		}

		if err := ds.risks.RemoveRisk(deleteRiskCtx, id, storage.RiskSubjectType_IMAGE); err != nil {
			return err
		}

		for _, imageComponent := range image.GetScan().GetComponents() {
			key := getImageComponentKey(imageComponent)
			ds.componentsInImages[key]--

			if ds.componentsInImages[key] == 0 {
				delete(ds.componentsInImages, key)

				if err := ds.risks.RemoveRisk(deleteRiskCtx, key, storage.RiskSubjectType_IMAGE_COMPONENT); err != nil {
					return err
				}
			}
		}

	}
	return errorList.ToError()
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := ds.canReadImage(ctx, id); err != nil || !ok {
		return false, err
	}
	return ds.storage.Exists(id)
}

// merge adds the most up to date data from the two inputs to the first input.
func merge(mergeTo *storage.Image, mergeWith *storage.Image) (updated bool) {
	// If the image currently in the DB has more up to date info, swap it out.
	if mergeWith.GetMetadata().GetV1().GetCreated().Compare(mergeTo.GetMetadata().GetV1().GetCreated()) >= 0 {
		mergeTo.Metadata = mergeWith.GetMetadata()
	} else {
		updated = true
	}
	if mergeWith.GetScan().GetScanTime().Compare(mergeTo.GetScan().GetScanTime()) >= 0 {
		mergeTo.Scan = mergeWith.GetScan()
	} else {
		updated = true
	}

	return
}

func (ds *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	log.Info("[STARTUP] Determining if image db/indexer reconciliation is needed")

	dbTxNum, err := ds.storage.GetTxnCount()
	if err != nil {
		return err
	}
	indexerTxNum := ds.indexer.GetTxnCount()

	if !txn.ReconciliationNeeded(dbTxNum, indexerTxNum) {
		log.Info("[STARTUP] Reconciliation for images is not needed")
		return nil
	}
	log.Info("[STARTUP] Indexing images")

	if err := ds.indexer.ResetIndex(); err != nil {
		return err
	}

	images, err := ds.storage.GetImages()
	if err != nil {
		return err
	}
	if err := ds.indexer.AddImages(images); err != nil {
		return err
	}

	if err := ds.storage.IncTxnCount(); err != nil {
		return err
	}
	if err := ds.indexer.SetTxnCount(dbTxNum + 1); err != nil {
		return err
	}

	log.Info("[STARTUP] Successfully indexed images")
	return nil
}

func (ds *datastoreImpl) updateListImagePriority(images ...*storage.ListImage) {
	for _, image := range images {
		image.Priority = ds.imageRanker.GetRankForID(image.GetId())
	}
}

func (ds *datastoreImpl) updateImagePriority(images ...*storage.Image) {
	for _, image := range images {
		for _, imageComponent := range image.GetScan().GetComponents() {
			imageComponent.Priority = ds.imageComponentRanker.GetRankForID(getImageComponentKey(imageComponent))
		}
		image.Priority = ds.imageRanker.GetRankForID(image.GetId())
	}
}

func getImageComponentKey(imageComponent *storage.EmbeddedImageScanComponent) string {
	return fmt.Sprintf("%s:%s", imageComponent.GetName(), imageComponent.GetVersion())
}
