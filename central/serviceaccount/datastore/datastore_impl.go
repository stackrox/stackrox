package datastore

import (
	"context"

	"github.com/pkg/errors"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/serviceaccount/internal/index"
	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	"github.com/stackrox/rox/central/serviceaccount/search"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	serviceAccountsSAC = sac.ForResource(resources.ServiceAccount)
	riskElevatedCtx    = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Risk),
		))
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
	risks    riskDS.DataStore
}

func (d *datastoreImpl) buildIndex() error {
	serviceAccounts, err := d.storage.ListServiceAccounts()
	if err != nil {
		return err
	}
	return d.indexer.AddServiceAccounts(serviceAccounts)
}

func (d *datastoreImpl) GetServiceAccount(ctx context.Context, id string) (*storage.ServiceAccount, bool, error) {
	acc, found, err := d.storage.GetServiceAccount(id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := serviceAccountsSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(acc).Allowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	return acc, true, nil
}

func (d *datastoreImpl) SearchRawServiceAccounts(ctx context.Context, q *v1.Query) ([]*storage.ServiceAccount, error) {
	return d.searcher.SearchRawServiceAccounts(ctx, q)
}

func (d *datastoreImpl) SearchServiceAccounts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return d.searcher.SearchServiceAccounts(ctx, q)
}

func (d *datastoreImpl) UpsertServiceAccount(ctx context.Context, request *storage.ServiceAccount) error {
	if ok, err := serviceAccountsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := d.storage.UpsertServiceAccount(request); err != nil {
		return err
	}
	return d.indexer.AddServiceAccount(request)
}

func (d *datastoreImpl) RemoveServiceAccount(ctx context.Context, id string) error {
	if ok, err := serviceAccountsSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := d.storage.DeleteServiceAccount(id); err != nil {
		return err
	}
	if err := d.risks.RemoveRisk(riskElevatedCtx, id, storage.RiskEntityType_SERVICEACCOUNT); err != nil {
		return err
	}
	return d.indexer.DeleteServiceAccount(id)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return d.searcher.Search(ctx, q)
}
