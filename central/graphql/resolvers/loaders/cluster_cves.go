package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var clusterCveLoaderType = reflect.TypeOf(storage.ClusterCVE{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.ClusterCVE{}), func() interface{} {
		return NewClusterCVELoader(clusterCVEDataStore.Singleton())
	})
}

// NewClusterCVELoader creates a new loader for cluster cve data.
func NewClusterCVELoader(ds clusterCVEDataStore.DataStore) ClusterCVELoader {
	return &clusterCveLoaderImpl{
		loaded: make(map[string]*storage.ClusterCVE),
		ds:     ds,
	}
}

// GetClusterCVELoader returns the ClusterCVELoader from the context if it exists.
func GetClusterCVELoader(ctx context.Context) (ClusterCVELoader, error) {
	loader, err := GetLoader(ctx, clusterCveLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(ClusterCVELoader), nil
}

// ClusterCVELoader loads cluster cve data, and stores already loaded cves for other ops in the same context to use.
type ClusterCVELoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.ClusterCVE, error)
	FromID(ctx context.Context, id string) (*storage.ClusterCVE, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ClusterCVE, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// clusterCveLoaderImpl implements the ClusterCVELoader interface.
type clusterCveLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.ClusterCVE

	ds clusterCVEDataStore.DataStore
}

// FromIDs loads a set of cluster cves from a set of ids.
func (idl *clusterCveLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.ClusterCVE, error) {
	cves, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

// FromID loads an cluster cve from an ID.
func (idl *clusterCveLoaderImpl) FromID(ctx context.Context, id string) (*storage.ClusterCVE, error) {
	cves, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return cves[0], nil
}

// FromQuery loads a set of cluster cves that match a query.
func (idl *clusterCveLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ClusterCVE, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

func (idl *clusterCveLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	count, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (idl *clusterCveLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.Count(ctx, search.EmptyQuery())
	return int32(count), err
}

func (idl *clusterCveLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.ClusterCVE, error) {
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

func (idl *clusterCveLoaderImpl) setAll(cves []*storage.ClusterCVE) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, cve := range cves {
		idl.loaded[cve.GetId()] = cve
	}
}

func (idl *clusterCveLoaderImpl) readAll(ids []string) (cves []*storage.ClusterCVE, missing []int) {
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
