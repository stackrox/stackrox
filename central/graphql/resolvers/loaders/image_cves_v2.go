package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	ImageCVEDataStore "github.com/stackrox/rox/central/cve/image/v2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

var imageCveV2LoaderType = reflect.TypeOf(storage.ImageCVEV2{})

func init() {
	RegisterTypeFactory(imageCveV2LoaderType, func() interface{} {
		return NewImageCVEV2Loader(ImageCVEDataStore.Singleton())
	})
}

// NewImageCVEV2Loader creates a new loader for image cve data.
func NewImageCVEV2Loader(ds ImageCVEDataStore.DataStore) ImageCVEV2Loader {
	return &imageCveV2LoaderImpl{
		loaded: make(map[string]*storage.ImageCVEV2),
		ds:     ds,
	}
}

// GetImageCVEV2Loader returns the ImageCVELoader from the context if it exists.
func GetImageCVEV2Loader(ctx context.Context) (ImageCVEV2Loader, error) {
	loader, err := GetLoader(ctx, imageCveV2LoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(ImageCVEV2Loader), nil
}

// ImageCVEV2Loader loads image cve data, and stores already loaded cves for other ops in the same context to use.
type ImageCVEV2Loader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.ImageCVEV2, error)
	FromID(ctx context.Context, id string) (*storage.ImageCVEV2, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ImageCVEV2, error)
	GetIDs(ctx context.Context, query *v1.Query) ([]string, error)
	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// imageCveV2LoaderImpl implements the ImageCVELoader interface.
type imageCveV2LoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.ImageCVEV2

	ds ImageCVEDataStore.DataStore
}

// FromIDs loads a set of image cves from a set of ids.
func (idl *imageCveV2LoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.ImageCVEV2, error) {
	cves, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

// FromID loads an image cve from an ID.
func (idl *imageCveV2LoaderImpl) FromID(ctx context.Context, id string) (*storage.ImageCVEV2, error) {
	cves, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return cves[0], nil
}

// FromQuery loads a set of image cves that match a query.
// NOTE: image_cves_v2 table has been dropped by migration m_222_to_m_223.
// CVE data is now stored in the normalized cves + component_cve_edges tables.
// This loader returns empty results until the GraphQL layer is updated.
func (idl *imageCveV2LoaderImpl) FromQuery(_ context.Context, _ *v1.Query) ([]*storage.ImageCVEV2, error) {
	return nil, nil
}

// GetIDs returns IDs matching a query.
// NOTE: returns empty pending migration to normalized CVE table.
func (idl *imageCveV2LoaderImpl) GetIDs(_ context.Context, _ *v1.Query) ([]string, error) {
	return nil, nil
}

// CountFromQuery returns count matching a query.
// NOTE: returns 0 pending migration to normalized CVE table.
func (idl *imageCveV2LoaderImpl) CountFromQuery(_ context.Context, _ *v1.Query) (int32, error) {
	return 0, nil
}

// CountAll returns total CVE count.
// NOTE: returns 0 pending migration to normalized CVE table.
func (idl *imageCveV2LoaderImpl) CountAll(_ context.Context) (int32, error) {
	return 0, nil
}

func (idl *imageCveV2LoaderImpl) load(_ context.Context, ids []string) ([]*storage.ImageCVEV2, error) {
	cves, missing := idl.readAll(ids)
	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all cves could be found: %s", strings.Join(missingIDs, ","))
	}
	return cves, nil
}

func (idl *imageCveV2LoaderImpl) setAll(cves []*storage.ImageCVEV2) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, cve := range cves {
		idl.loaded[cve.GetId()] = cve
	}
}

func (idl *imageCveV2LoaderImpl) readAll(ids []string) (cves []*storage.ImageCVEV2, missing []int) {
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
