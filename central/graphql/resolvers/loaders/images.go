package loaders

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var imageLoaderType = reflect.TypeOf(storage.Image{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.Image{}), func() interface{} {
		return NewImageLoader(datastore.Singleton())
	})
}

// NewImageLoader creates a new loader for image data. If postgres is enabled, this loader holds images without scan data—components and vulns.
func NewImageLoader(ds datastore.DataStore) ImageLoader {
	return &imageLoaderImpl{
		loaded: make(map[string]*storage.Image),
		ds:     ds,
	}
}

// GetImageLoader returns the ImageLoader from the context if it exists.
func GetImageLoader(ctx context.Context) (ImageLoader, error) {
	loader, err := GetLoader(ctx, imageLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(ImageLoader), nil
}

// ImageLoader loads image data, and stores already loaded images for other ops in the same context to use.
type ImageLoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.Image, error)
	FromID(ctx context.Context, id string) (*storage.Image, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Image, error)
	FullImageWithID(ctx context.Context, id string) (*storage.Image, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// imageLoaderImpl implements the ImageDataLoader interface.
type imageLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.Image

	ds datastore.DataStore
}

// FromIDs loads a set of images from a set of ids.
func (idl *imageLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.Image, error) {
	images, err := idl.load(ctx, ids, false)
	if err != nil {
		return nil, err
	}
	return images, nil
}

// FromID loads an image from an ID, without scan components and vulns.
func (idl *imageLoaderImpl) FromID(ctx context.Context, id string) (*storage.Image, error) {
	images, err := idl.load(ctx, []string{id}, false)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, errors.Errorf("could not find image for id %q:", id)
	}
	return images[0], nil
}

// FullImageWithID loads full image from an ID.
func (idl *imageLoaderImpl) FullImageWithID(ctx context.Context, id string) (*storage.Image, error) {
	image, err := idl.FromID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Load the full image if full scan is not available.
	if image.GetComponents() == 0 || len(image.GetScan().GetComponents()) > 0 {
		return image, nil
	}

	idl.lock.Lock()
	delete(idl.loaded, id)
	idl.lock.Unlock()

	images, err := idl.load(ctx, []string{id}, true)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, errors.Errorf("could not find image for id %q:", id)
	}
	return images[0], nil
}

// FromQuery loads a set of images that match a query.
func (idl *imageLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Image, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

func (idl *imageLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	count, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (idl *imageLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.CountImages(ctx)
	return int32(count), err
}

func (idl *imageLoaderImpl) load(ctx context.Context, ids []string, pullFullObject bool) ([]*storage.Image, error) {
	images, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
		// `pullFullObject` is only supported on Postgres.
		if pullFullObject {
			images, err = idl.ds.GetImagesBatch(ctx, collectMissing(ids, missing))
		} else {
			images, err = idl.ds.GetManyImageMetadata(ctx, collectMissing(ids, missing))
		}
		if err != nil {
			return nil, err
		}
		idl.setAll(images)
		images, _ = idl.readAll(ids)
	}
	return images, nil
}

func (idl *imageLoaderImpl) setAll(images []*storage.Image) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, image := range images {
		idl.loaded[image.GetId()] = image
	}
}

func (idl *imageLoaderImpl) readAll(ids []string) (images []*storage.Image, missing []int) {
	idl.lock.RLock()
	defer idl.lock.RUnlock()

	for idx, id := range ids {
		image, isLoaded := idl.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			images = append(images, image)
		}
	}
	return
}
