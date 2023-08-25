package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	ImageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var imageCveLoaderType = reflect.TypeOf(storage.ImageCVE{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.ImageCVE{}), func() interface{} {
		return NewImageCVELoader(ImageCVEDataStore.Singleton())
	})
}

// NewImageCVELoader creates a new loader for image cve data.
func NewImageCVELoader(ds ImageCVEDataStore.DataStore) ImageCVELoader {
	return &imageCveLoaderImpl{
		loaded: make(map[string]*storage.ImageCVE),
		ds:     ds,
	}
}

// GetImageCVELoader returns the ImageCVELoader from the context if it exists.
func GetImageCVELoader(ctx context.Context) (ImageCVELoader, error) {
	loader, err := GetLoader(ctx, imageCveLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(ImageCVELoader), nil
}

// ImageCVELoader loads image cve data, and stores already loaded cves for other ops in the same context to use.
type ImageCVELoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.ImageCVE, error)
	FromID(ctx context.Context, id string) (*storage.ImageCVE, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageCVE, error)
	GetIDs(ctx context.Context, query *v1.Query) ([]string, error)
	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// imageCveLoaderImpl implements the ImageCVELoader interface.
type imageCveLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.ImageCVE

	ds ImageCVEDataStore.DataStore
}

// FromIDs loads a set of image cves from a set of ids.
func (idl *imageCveLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.ImageCVE, error) {
	cves, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

// FromID loads an image cve from an ID.
func (idl *imageCveLoaderImpl) FromID(ctx context.Context, id string) (*storage.ImageCVE, error) {
	cves, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return cves[0], nil
}

// FromQuery loads a set of image cves that match a query.
func (idl *imageCveLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageCVE, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

func (idl *imageCveLoaderImpl) GetIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return search.ResultsToIDs(results), nil
}

func (idl *imageCveLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	count, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (idl *imageCveLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.Count(ctx, search.EmptyQuery())
	return int32(count), err
}

func (idl *imageCveLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.ImageCVE, error) {
	cves, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
		cves, err = idl.ds.GetBatch(ctx, collectMissing(ids, missing))
		if err != nil {
			return nil, err
		}
		idl.setAll(cves)
		cves, missing = idl.readAll(ids)
	}
	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all cves could be found: %s", strings.Join(missingIDs, ","))
	}
	return cves, nil
}

func (idl *imageCveLoaderImpl) setAll(cves []*storage.ImageCVE) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, cve := range cves {
		idl.loaded[cve.GetId()] = cve
	}
}

func (idl *imageCveLoaderImpl) readAll(ids []string) (cves []*storage.ImageCVE, missing []int) {
	idl.lock.RLock()
	defer idl.lock.RUnlock()

	for idx, id := range ids {
		cve, isLoaded := idl.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			cves = append(cves, cve)
		}
	}
	return
}
