package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/deployment/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var deploymentLoaderType = reflect.TypeOf(storage.Deployment{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.Deployment{}), func() interface{} {
		return NewDeploymentLoader(datastore.Singleton())
	})
}

// NewDeploymentLoader creates a new loader for deployment data.
func NewDeploymentLoader(ds datastore.DataStore) DeploymentLoader {
	return &deploymentLoaderImpl{
		loaded: make(map[string]*storage.Deployment),
		ds:     ds,
	}
}

// GetDeploymentLoader returns the DeploymentLoader from the context if it exists.
func GetDeploymentLoader(ctx context.Context) (DeploymentLoader, error) {
	loader, err := GetLoader(ctx, deploymentLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(DeploymentLoader), nil
}

// DeploymentLoader loads deployment data, and stores already loaded deployments for other ops in the same context to use.
type DeploymentLoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.Deployment, error)
	FromID(ctx context.Context, id string) (*storage.Deployment, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Deployment, error)

	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// deploymentLoaderImpl implements the DeploymentDataLoader interface.
type deploymentLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.Deployment

	ds datastore.DataStore
}

// FromIDs loads a set of deployments from a set of ids.
func (idl *deploymentLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.Deployment, error) {
	deployments, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

// FromID loads an deployment from an ID.
func (idl *deploymentLoaderImpl) FromID(ctx context.Context, id string) (*storage.Deployment, error) {
	deployments, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return deployments[0], nil
}

// FromQuery loads a set of deployments that match a query.
func (idl *deploymentLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Deployment, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

// CountFromQuery returns the number of deployments that match a given query.
func (idl *deploymentLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	count, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// CountFromQuery returns the total number of deployments.
func (idl *deploymentLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.CountDeployments(ctx)
	return int32(count), err
}

func (idl *deploymentLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.Deployment, error) {
	deployments, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
		deployments, err = idl.ds.GetDeployments(ctx, collectMissing(ids, missing))
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
		return nil, errors.Errorf("not all deployments could be found: %s", strings.Join(missingIDs, ","))
	}
	return deployments, nil
}

func (idl *deploymentLoaderImpl) setAll(deployments []*storage.Deployment) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, deployment := range deployments {
		idl.loaded[deployment.GetId()] = deployment
	}
}

func (idl *deploymentLoaderImpl) readAll(ids []string) (deployments []*storage.Deployment, missing []int) {
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
