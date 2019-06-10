package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/txn"
)

var (
	log = logging.LoggerForModule()

	imagesSAC = sac.ForResource(resources.Image)
)

type datastoreImpl struct {
	lock sync.Mutex

	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func newDatastoreImpl(storage store.Store, indexer index.Indexer, searcher search.Searcher) (*datastoreImpl, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
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
	scrubClusterNSScopes(imgs...)
	return imgs, nil
}

func (ds *datastoreImpl) SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error) {
	imgs, err := ds.searcher.SearchListImages(ctx, q)
	if err != nil {
		return nil, err
	}
	scrubClusterNSScopesFromListImages(imgs...)
	return imgs, nil
}

func (ds *datastoreImpl) ListImage(ctx context.Context, sha string) (*storage.ListImage, bool, error) {
	img, found, err := ds.storage.ListImage(sha)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := imagesSAC.ReadAllowedForClusterNSScopes(ctx, img.GetClusternsScopes()); err != nil || !ok {
		return nil, false, err
	}

	scrubClusterNSScopesFromListImages(img)
	return img, true, nil
}

func (ds *datastoreImpl) ListImages(ctx context.Context) ([]*storage.ListImage, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if ok {
		imgs, err := ds.storage.ListImages()
		if err != nil {
			return nil, err
		}
		scrubClusterNSScopesFromListImages(imgs...)
		return imgs, nil
	}

	return ds.SearchListImages(ctx, searchPkg.EmptyQuery())
}

// GetImages delegates to the underlying store.
func (ds *datastoreImpl) GetImages(ctx context.Context) ([]*storage.Image, error) {
	if ok, sacErr := imagesSAC.ReadAllowed(ctx); sacErr != nil {
		return nil, sacErr
	} else if ok {
		images, err := ds.storage.GetImages()
		if err != nil {
			return nil, err
		}
		scrubClusterNSScopes(images...)
		return images, nil
	}

	return ds.SearchRawImages(ctx, searchPkg.EmptyQuery())
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

// GetImage delegates to the underlying store.
func (ds *datastoreImpl) GetImage(ctx context.Context, sha string) (*storage.Image, bool, error) {
	img, found, err := ds.storage.GetImage(sha)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := imagesSAC.ReadAllowedForClusterNSScopes(ctx, img.GetClusternsScopes()); err != nil || !ok {
		return nil, false, err
	}

	scrubClusterNSScopes(img)
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
		scrubClusterNSScopes(imgs...)
		return imgs, nil
	}

	shasQuery := searchPkg.NewQueryBuilder().AddStrings(searchPkg.ImageSHA, shas...).ProtoQuery()
	return ds.SearchRawImages(ctx, shasQuery)
}

// UpsertImage dedupes the image with the underlying storage and adds the image to the index.
func (ds *datastoreImpl) UpsertImage(ctx context.Context, image *storage.Image) error {
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	oldImage, exists, err := ds.storage.GetImage(image.GetId())
	if err != nil {
		return err
	}
	if exists {
		merge(image, oldImage)
	}
	if err = ds.storage.UpsertImage(image); err != nil {
		return err
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
	for _, id := range ids {
		if err := ds.storage.DeleteImage(id); err != nil {
			errorList.AddError(err)
			continue
		}
		if err := ds.indexer.DeleteImage(id); err != nil {
			errorList.AddError(err)
		}
	}
	return errorList.ToError()
}

func scrubClusterNSScopes(images ...*storage.Image) {
	for _, img := range images {
		img.ClusternsScopes = nil
	}
}

func scrubClusterNSScopesFromListImages(images ...*storage.ListImage) {
	for _, img := range images {
		img.ClusternsScopes = nil
	}
}

// merge adds the most up to date data from the two inputs to the first input.
func merge(mergeTo *storage.Image, mergeWith *storage.Image) {
	// If the image currently in the DB has more up to date info, swap it out.
	if mergeWith.GetMetadata().GetV1().GetCreated().Compare(mergeTo.GetMetadata().GetV1().GetCreated()) > 0 {
		mergeTo.Metadata = mergeWith.GetMetadata()
	}
	if mergeWith.GetScan().GetScanTime().Compare(mergeTo.GetScan().GetScanTime()) > 0 {
		mergeTo.Scan = mergeWith.GetScan()
	}

	if len(mergeTo.GetClusternsScopes()) == 0 {
		mergeTo.ClusternsScopes = mergeWith.GetClusternsScopes()
	} else {
		for k, v := range mergeWith.GetClusternsScopes() {
			mergeTo.ClusternsScopes[k] = v
		}
	}
}

func (ds *datastoreImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	log.Infof("[STARTUP] Determining if image db/indexer reconciliation is needed")

	dbTxNum, err := ds.storage.GetTxnCount()
	if err != nil {
		return err
	}
	indexerTxNum := ds.indexer.GetTxnCount()

	if !txn.ReconciliationNeeded(dbTxNum, indexerTxNum) {
		log.Infof("[STARTUP] Reconciliation for images is not needed")
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
