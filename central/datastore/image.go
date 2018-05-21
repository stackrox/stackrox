package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// ImageDataStore is an intermediary to ImageStorage.
type ImageDataStore interface {
	// This is an embedded type so we don't have to override all functions. Indexing is a subset of Storage
	db.ImageStorage

	SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error)
}

// NewImageDataStore provides a new instance of ImageDataStore
func NewImageDataStore(storage db.ImageStorage, indexer search.ImageIndex) (ImageDataStore, error) {
	ds := &imageDataStoreImpl{
		ImageStorage: storage,
		indexer:      indexer,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

// ImageDataStore provides an intermediary implementation layer for ImageStorage.
type imageDataStoreImpl struct {
	// This is an embedded type so we don't have to override all functions. Indexing is a subset of Storage
	db.ImageStorage

	indexer search.ImageIndex
}

func (ds *imageDataStoreImpl) buildIndex() error {
	images, err := ds.GetImages()
	if err != nil {
		return err
	}
	for _, i := range images {
		if err := ds.indexer.AddImage(i); err != nil {
			logger.Errorf("Error inserting image %s (%s) into index: %s", i.GetName().GetSha(), i.GetName().GetFullName(), err)
		}
	}
	return nil
}

// SearchImages retrieves SearchResults from the indexer and storage
func (ds *imageDataStoreImpl) SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	images, results, err := ds.searchImages(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(images))
	for i, image := range images {
		protoResults = append(protoResults, search.ConvertImage(image, results[i]))
	}
	return protoResults, nil
}

// SearchRawImages retrieves SearchResults from the indexer and storage
func (ds *imageDataStoreImpl) SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error) {
	images, _, err := ds.searchImages(request)
	return images, err
}

// AddImage adds an image to the storage and the indexer
func (ds *imageDataStoreImpl) AddImage(image *v1.Image) error {
	if err := ds.ImageStorage.AddImage(image); err != nil {
		return err
	}
	return ds.indexer.AddImage(image)
}

// UpdateImage updates an image in storage and the indexer
func (ds *imageDataStoreImpl) UpdateImage(image *v1.Image) error {
	if err := ds.ImageStorage.UpdateImage(image); err != nil {
		return err
	}
	return ds.indexer.AddImage(image)
}

// RemoveImage removes an image from storage and the indexer
func (ds *imageDataStoreImpl) RemoveImage(id string) error {
	if err := ds.ImageStorage.RemoveImage(id); err != nil {
		return err
	}
	return ds.indexer.DeleteImage(id)
}

func (ds *imageDataStoreImpl) searchImages(request *v1.ParsedSearchRequest) ([]*v1.Image, []search.Result, error) {
	results, err := ds.indexer.SearchImages(request)
	if err != nil {
		return nil, nil, err
	}
	var images []*v1.Image
	var newResults []search.Result
	for _, result := range results {
		image, exists, err := ds.GetImage(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		images = append(images, image)
		newResults = append(newResults, result)
	}
	return images, newResults, nil
}
