package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/deployment/datastore"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var deploymentLoaderType = reflect.TypeOf(storage.Deployment{})

func init() {
	RegisterTypeFactory(reflect.TypeOf(storage.Deployment{}), func() interface{} {
		return NewDeploymentLoader(datastore.Singleton(), deploymentsView.Singleton())
	})
}

// NewDeploymentLoader creates a new loader for deployment data.
func NewDeploymentLoader(ds datastore.DataStore, deploymentView deploymentsView.DeploymentView) DeploymentLoader {
	return &deploymentLoaderImpl{
		loaded:         make(map[string]*storage.Deployment),
		ds:             ds,
		deploymentView: deploymentView,
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

	ds             datastore.DataStore
	deploymentView deploymentsView.DeploymentView
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

// queryContainsLifecycleStage recursively checks if a query contains a Lifecycle Stage filter.
func queryContainsLifecycleStage(query *v1.Query) bool {
	if query == nil {
		return false
	}

	// Check base query for Lifecycle Stage field.
	if baseQuery := query.GetBaseQuery(); baseQuery != nil {
		if baseQuery.GetMatchFieldQuery() != nil && baseQuery.GetMatchFieldQuery().GetField() == search.LifecycleStage.String() {
			return true
		}
	}

	// Recursively check conjunction queries.
	if conjunction := query.GetConjunction(); conjunction != nil {
		for _, q := range conjunction.GetQueries() {
			if queryContainsLifecycleStage(q) {
				return true
			}
		}
	}

	// Recursively check disjunction queries.
	if disjunction := query.GetDisjunction(); disjunction != nil {
		for _, q := range disjunction.GetQueries() {
			if queryContainsLifecycleStage(q) {
				return true
			}
		}
	}

	return false
}

// ensureLifecycleStageFilter adds a default filter for lifecycle_stage = ACTIVE if not already specified.
// This ensures backward compatibility by excluding soft-deleted deployments from GraphQL queries by default.
// If the query already contains a lifecycle_stage filter, the default is not added (user's explicit choice takes precedence).
func ensureLifecycleStageFilter(query *v1.Query) *v1.Query {
	// If query already has a lifecycle_stage filter, don't add default.
	if queryContainsLifecycleStage(query) {
		return query
	}

	// Add default filter: lifecycle_stage = ACTIVE.
	lifecycleFilter := search.NewQueryBuilder().
		AddStrings(search.LifecycleStage, storage.DeploymentLifecycleStage_DEPLOYMENT_ACTIVE.String()).
		ProtoQuery()

	if query == nil {
		return lifecycleFilter
	}

	// Combine user query with lifecycle filter.
	return search.ConjunctionQuery(query, lifecycleFilter)
}

// FromQuery loads a set of deployments that match a query.
// By default, only active deployments (lifecycle_stage = ACTIVE) are returned.
func (idl *deploymentLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.Deployment, error) {
	filteredQuery := ensureLifecycleStageFilter(query)
	responses, err := idl.deploymentView.Get(ctx, filteredQuery)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, responsesToDeploymentIDs(responses))
}

// CountFromQuery returns the number of deployments that match a given query.
// By default, only active deployments (lifecycle_stage = ACTIVE) are counted.
func (idl *deploymentLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	filteredQuery := ensureLifecycleStageFilter(query)
	count, err := idl.ds.Count(ctx, filteredQuery)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// CountAll returns the total number of active deployments (excludes soft-deleted).
func (idl *deploymentLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	// Use CountFromQuery with nil query to apply default lifecycle_stage filter.
	return idl.CountFromQuery(ctx, nil)
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

func responsesToDeploymentIDs(responses []deploymentsView.DeploymentCore) []string {
	ids := make([]string, 0, len(responses))
	for _, r := range responses {
		ids = append(ids, r.GetDeploymentID())
	}
	return ids
}
