package loaders

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imagev2/datastore"
	imagesView "github.com/stackrox/rox/central/views/images"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

// DO NOT SUBMIT: fix callers to work with a pointer (go/goprotoapi-findings#message-value)
var imageV2LoaderType = reflect.TypeOf(&storage.ImageV2{})

func init() {
	// DO NOT SUBMIT: fix callers to work with a pointer (go/goprotoapi-findings#message-value)
	RegisterTypeFactory(reflect.TypeOf(&storage.ImageV2{}), func() interface{} {
		return NewImageV2Loader(datastore.Singleton(), imagesView.Singleton())
	})
}

// NewImageV2Loader creates a new loader for image data.
func NewImageV2Loader(ds datastore.DataStore, imageView imagesView.ImageView) ImageV2Loader {
	return &imageV2LoaderImpl{
		loaded:    make(map[string]*storage.ImageV2),
		ds:        ds,
		imageView: imageView,
	}
}

// GetImageV2Loader returns the ImageLoader from the context if it exists.
func GetImageV2Loader(ctx context.Context) (ImageV2Loader, error) {
	loader, err := GetLoader(ctx, imageV2LoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(ImageV2Loader), nil
}

// ImageV2Loader loads image data, and stores already loaded images for other ops in the same context to use.
type ImageV2Loader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.ImageV2, error)
	FromID(ctx context.Context, id string) (*storage.ImageV2, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageV2, error)
	FullImageWithID(ctx context.Context, id string) (*storage.ImageV2, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// imageV2LoaderImpl implements the ImageDataLoader interface.
type imageV2LoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.ImageV2

	ds        datastore.DataStore
	imageView imagesView.ImageView
}

// FromIDs loads a set of images from a set of ids.
func (idl *imageV2LoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.ImageV2, error) {
	images, err := idl.load(ctx, ids, false)
	if err != nil {
		return nil, err
	}
	return images, nil
}

// FromID loads an image from an ID, without scan components and vulns.
func (idl *imageV2LoaderImpl) FromID(ctx context.Context, id string) (*storage.ImageV2, error) {
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
func (idl *imageV2LoaderImpl) FullImageWithID(ctx context.Context, id string) (*storage.ImageV2, error) {
	image, err := idl.FromID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Load the full image if full scan is not available.
	if image.GetScanStats().GetComponentCount() == 0 || len(image.GetScan().GetComponents()) > 0 {
		return image, nil
	}

	concurrency.WithLock(&idl.lock, func() {
		delete(idl.loaded, id)
	})

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
func (idl *imageV2LoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageV2, error) {
	responses, err := idl.imageView.Get(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, responsesToImageIDs(responses))
}

func (idl *imageV2LoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	count, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (idl *imageV2LoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.Count(ctx, searchPkg.EmptyQuery())
	return int32(count), err
}

func (idl *imageV2LoaderImpl) load(ctx context.Context, ids []string, pullFullObject bool) ([]*storage.ImageV2, error) {
	images, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
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

func (idl *imageV2LoaderImpl) setAll(images []*storage.ImageV2) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, image := range images {
		idl.loaded[image.GetId()] = image
	}
}

func (idl *imageV2LoaderImpl) readAll(ids []string) (images []*storage.ImageV2, missing []int) {
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
