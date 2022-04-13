package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/deployment/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/sync"
)

var listDeploymentLoaderType = reflect.TypeOf(storage.ListDeployment{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.ListDeployment{}), func() interface{} {
		return NewListDeploymentLoader(datastore.Singleton())
	})
}

// NewListDeploymentLoader creates a new loader for deployment data.
func NewListDeploymentLoader(ds datastore.DataStore) ListDeploymentLoader {
	return &listDeploymentLoaderImpl{
		loaded: make(map[string]*storage.ListDeployment),
		ds:     ds,
	}
}

// GetListDeploymentLoader returns the DeploymentLoader from the context if it exists.
func GetListDeploymentLoader(ctx context.Context) (ListDeploymentLoader, error) {
	loader, err := GetLoader(ctx, listDeploymentLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(ListDeploymentLoader), nil
}

// ListDeploymentLoader loads deployment data, and stores already loaded deployments for other ops in the same context to use.
type ListDeploymentLoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.ListDeployment, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ListDeployment, error)
}

// listDeploymentLoaderImpl implements the ListsDeploymentDataLoader interface.
type listDeploymentLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.ListDeployment

	ds datastore.DataStore
}

// FromQuery loads a set of deployments that match a query.
func (idl *listDeploymentLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.ListDeployment, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

// FromIDs returns the list deployments for a list of ids.
func (idl *listDeploymentLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.ListDeployment, error) {
	deployments, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func (idl *listDeploymentLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.ListDeployment, error) {
	deployments, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
		deployments, err = idl.ds.SearchListDeployments(ctx, search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery())
		if err != nil {
			return nil, err
		}
		idl.setAll(deployments)
		deployments, missing = idl.readAll(ids)
	}
	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all list deployments could be found: %s", strings.Join(missingIDs, ","))
	}
	return deployments, nil
}

func (idl *listDeploymentLoaderImpl) setAll(deployments []*storage.ListDeployment) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, deployment := range deployments {
		idl.loaded[deployment.GetId()] = deployment
	}
}

func (idl *listDeploymentLoaderImpl) readAll(ids []string) (deployments []*storage.ListDeployment, missing []int) {
	idl.lock.RLock()
	defer idl.lock.RUnlock()

	for idx, id := range ids {
		deployment, isLoaded := idl.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			deployments = append(deployments, deployment)
		}
	}
	return
}
