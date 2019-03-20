package datastore

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/image/search"
	"github.com/stackrox/rox/central/image/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	lock sync.RWMutex

	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *datastoreImpl) Search(q *v1.Query) ([]searchPkg.Result, error) {
	return ds.indexer.Search(q)
}

func (ds *datastoreImpl) SearchImages(q *v1.Query) ([]*v1.SearchResult, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.searcher.SearchImages(q)
}

// SearchRawImages delegates to the underlying searcher.
func (ds *datastoreImpl) SearchRawImages(q *v1.Query) ([]*storage.Image, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.searcher.SearchRawImages(q)
}

func (ds *datastoreImpl) SearchListImages(q *v1.Query) ([]*storage.ListImage, error) {
	return ds.searcher.SearchListImages(q)
}

func (ds *datastoreImpl) ListImage(sha string) (*storage.ListImage, bool, error) {
	return ds.storage.ListImage(sha)
}

func (ds *datastoreImpl) ListImages() ([]*storage.ListImage, error) {
	return ds.storage.ListImages()
}

// GetImages delegates to the underlying store.
func (ds *datastoreImpl) GetImages() ([]*storage.Image, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.storage.GetImages()
}

// CountImages delegates to the underlying store.
func (ds *datastoreImpl) CountImages() (int, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.storage.CountImages()
}

// GetImage delegates to the underlying store.
func (ds *datastoreImpl) GetImage(sha string) (*storage.Image, bool, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.storage.GetImage(sha)
}

// GetImagesBatch delegates to the underlying store.
func (ds *datastoreImpl) GetImagesBatch(shas []string) ([]*storage.Image, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.storage.GetImagesBatch(shas)
}

// UpsertImage dedupes the image with the underlying storage and adds the image to the index.
func (ds *datastoreImpl) UpsertImage(image *storage.Image) error {
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

// merge adds the most up to date data from the two inputs to the first input.
func merge(mergeTo *storage.Image, mergeWith *storage.Image) {
	// If the image currently in the DB has more up to date info, swap it out.
	if protoconv.CompareProtoTimestamps(mergeWith.GetMetadata().GetV1().GetCreated(), mergeTo.GetMetadata().GetV1().GetCreated()) > 0 {
		mergeTo.Metadata = mergeWith.GetMetadata()
	}
	if protoconv.CompareProtoTimestamps(mergeWith.GetScan().GetScanTime(), mergeTo.GetScan().GetScanTime()) > 0 {
		mergeTo.Scan = mergeWith.GetScan()
	}
}
