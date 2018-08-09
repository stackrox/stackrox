package datastore

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/image/search"
	"github.com/stackrox/rox/central/image/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
)

type datastoreImpl struct {
	lock sync.RWMutex

	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *datastoreImpl) SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.searcher.SearchImages(request)
}

// SearchRawImages delegates to the underlying searcher.
func (ds *datastoreImpl) SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.searcher.SearchRawImages(request)
}

func (ds *datastoreImpl) SearchListImages(request *v1.ParsedSearchRequest) ([]*v1.ListImage, error) {
	return ds.searcher.SearchListImages(request)
}

func (ds *datastoreImpl) ListImage(sha string) (*v1.ListImage, bool, error) {
	return ds.storage.ListImage(sha)
}

func (ds *datastoreImpl) ListImages() ([]*v1.ListImage, error) {
	return ds.storage.ListImages()
}

// GetImages delegates to the underlying store.
func (ds *datastoreImpl) GetImages() ([]*v1.Image, error) {
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
func (ds *datastoreImpl) GetImage(sha string) (*v1.Image, bool, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.storage.GetImage(sha)
}

// GetImagesBatch delegates to the underlying store.
func (ds *datastoreImpl) GetImagesBatch(shas []string) ([]*v1.Image, error) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	return ds.storage.GetImagesBatch(shas)
}

// UpsertDedupeImage dedupes the image with the underlying storage and adds the image to the index.
func (ds *datastoreImpl) UpsertDedupeImage(image *v1.Image) error {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Check the images sha, and figure out if it needs to be changed.
	shaIs := image.GetName().GetSha()
	if shaIs == "" {
		return fmt.Errorf("cannot process an image with no sha: %s", image.GetName())
	}
	shaShouldBe, err := ds.dedupeSha(image)
	if err != nil {
		return err
	}

	// Fetch the current image in the DB, and use any information that is more up to date.
	err = ds.mergeWithDbVersion(image, shaShouldBe)
	if err != nil {
		return err
	}

	// If the sha should be changed, we need to merge in the old data, and remove old index and
	// storage data.
	if shaIs != shaShouldBe {
		err = ds.mergeWithDbVersion(image, shaIs)
		if err != nil {
			return err
		}
		err = ds.handleShaChange(shaIs, shaShouldBe)
		if err != nil {
			return err
		}
	}

	// Finally upsert the new image to the store and index.
	image.GetName().Sha = shaShouldBe
	if err = ds.storage.UpsertImage(image); err != nil {
		return err
	}
	return ds.indexer.AddImage(image)
}

// dedupeSha looks for what the sha should be for the given image, first by checking the metadata in the image
// itself, then by looking for a redirect in the store.
func (ds *datastoreImpl) dedupeSha(image *v1.Image) (shaShouldBe string, err error) {
	// Check the images sha.
	shaShouldBe = regShaForimage(image)
	if shaShouldBe != "" {
		return
	}

	// Otherwise, load if we have a registry sha that matches already.
	shaIs := image.GetName().GetSha()
	var exists bool
	shaShouldBe, exists, err = ds.storage.GetRegistrySha(shaIs)
	if err != nil {
		return
	}
	if exists {
		return
	}
	shaShouldBe = shaIs
	return
}

// mergeWithDbVersion adds the data for the given id to the given image if it is more up to date.
func (ds *datastoreImpl) mergeWithDbVersion(image *v1.Image, shaShouldBe string) (err error) {
	// Fetch the current image in the DB, and use any information that is more up to date.
	oldImage, exists, err := ds.storage.GetImage(shaShouldBe)
	if err != nil || !exists {
		return
	}
	merge(image, oldImage)
	return
}

// handleShaChange adds the sha redirect to the store and removes the old sha's data.
func (ds *datastoreImpl) handleShaChange(oldSha, newSha string) (err error) {
	err = ds.storage.UpsertRegistrySha(oldSha, newSha)
	if err != nil {
		return
	}
	err = ds.storage.DeleteImage(oldSha)
	if err != nil {
		return
	}
	err = ds.indexer.DeleteImage(oldSha)
	return
}

// merge adds the most up to date data from the two inputs to the first input.
func merge(mergeTo *v1.Image, mergeWith *v1.Image) {
	// If the image currently in the DB has more up to date info, swap it out.
	if protoconv.CompareProtoTimestamps(mergeWith.GetMetadata().GetCreated(), mergeTo.GetMetadata().GetCreated()) > 0 {
		mergeTo.Metadata = mergeWith.GetMetadata()
	}
	if protoconv.CompareProtoTimestamps(mergeWith.GetScan().GetScanTime(), mergeTo.GetScan().GetScanTime()) > 0 {
		mergeTo.Scan = mergeWith.GetScan()
	}
}

func regShaForimage(image *v1.Image) string {
	if image.GetMetadata().GetRegistrySha() != "" {
		return image.GetMetadata().GetRegistrySha()
	} else if image.GetMetadata().GetV2().GetDigest() != "" {
		return image.GetMetadata().GetV2().GetDigest()
	}
	return ""
}
