package datastore

import (
	"context"

	"github.com/stackrox/rox/central/deployment/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// activeStateDatastore wraps a DataStore and transparently injects an
// active-deployment filter into every read operation. Mutation methods and
// GetImagesForDeployment delegate directly to the inner datastore.
type activeStateDatastore struct {
	ds DataStore
}

// NewActiveStateDatastore returns a DataStore that only exposes active
// deployments. All read methods automatically conjunct the caller's
// query with ActiveDeploymentsQuery().
func NewActiveStateDatastore(ds DataStore) DataStore {
	return &activeStateDatastore{ds: ds}
}

// activeQuery returns the caller's query conjuncted with the active-deployment filter.
func activeQuery(q *v1.Query) *v1.Query {
	return pkgSearch.ConjunctionQuery(q, ActiveDeploymentsQuery())
}

// --- Query-based methods ---

func (a *activeStateDatastore) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return a.ds.Search(ctx, activeQuery(q))
}

func (a *activeStateDatastore) Count(ctx context.Context, q *v1.Query) (int, error) {
	return a.ds.Count(ctx, activeQuery(q))
}

func (a *activeStateDatastore) SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return a.ds.SearchDeployments(ctx, activeQuery(q))
}

func (a *activeStateDatastore) SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	return a.ds.SearchRawDeployments(ctx, activeQuery(q))
}

func (a *activeStateDatastore) SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error) {
	return a.ds.SearchListDeployments(ctx, activeQuery(q))
}

func (a *activeStateDatastore) GetDeploymentIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	return a.ds.GetDeploymentIDs(ctx, activeQuery(q))
}

func (a *activeStateDatastore) WalkByQuery(ctx context.Context, query *v1.Query, fn func(deployment *storage.Deployment) error) error {
	return a.ds.WalkByQuery(ctx, activeQuery(query), fn)
}

func (a *activeStateDatastore) GetContainerImageViews(ctx context.Context, q *v1.Query) ([]*views.ContainerImageView, error) {
	return a.ds.GetContainerImageViews(ctx, activeQuery(q))
}

// --- ID-based methods ---

func (a *activeStateDatastore) GetDeployment(ctx context.Context, id string) (*storage.Deployment, bool, error) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, id).ProtoQuery()
	results, err := a.SearchRawDeployments(ctx, q)
	if err != nil || len(results) == 0 {
		return nil, false, err
	}
	return results[0], true, nil
}

func (a *activeStateDatastore) GetDeployments(ctx context.Context, ids []string) ([]*storage.Deployment, error) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, ids...).ProtoQuery()
	return a.SearchRawDeployments(ctx, q)
}

func (a *activeStateDatastore) ListDeployment(ctx context.Context, id string) (*storage.ListDeployment, bool, error) {
	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, id).ProtoQuery()
	results, err := a.SearchListDeployments(ctx, q)
	if err != nil || len(results) == 0 {
		return nil, false, err
	}
	return results[0], true, nil
}

// --- Pass-through methods ---

func (a *activeStateDatastore) UpsertDeployment(ctx context.Context, deployment *storage.Deployment) error {
	return a.ds.UpsertDeployment(ctx, deployment)
}

func (a *activeStateDatastore) RemoveDeployment(ctx context.Context, clusterID, id string) error {
	return a.ds.RemoveDeployment(ctx, clusterID, id)
}

func (a *activeStateDatastore) GetImagesForDeployment(ctx context.Context, deployment *storage.Deployment) ([]*storage.Image, error) {
	return a.ds.GetImagesForDeployment(ctx, deployment)
}

func (a *activeStateDatastore) PurgeDeployments(ctx context.Context, q *v1.Query) (int, error) {
	return a.ds.PurgeDeployments(ctx, q)
}
