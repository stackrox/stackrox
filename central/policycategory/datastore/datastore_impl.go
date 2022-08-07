package datastore

import (
	"context"
	"fmt"
	"strings"

	errorsPkg "github.com/pkg/errors"
	"github.com/stackrox/rox/central/policycategory/index"
	"github.com/stackrox/rox/central/policycategory/search"
	"github.com/stackrox/rox/central/policycategory/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log               = logging.LoggerForModule()
	policyCategorySAC = sac.ForResource(resources.Policy)

	policyCategoryCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher

	categoryMutex sync.Mutex
}

func (ds *datastoreImpl) buildIndex() error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	var categories []*storage.PolicyCategory
	err := ds.storage.Walk(policyCategoryCtx, func(category *storage.PolicyCategory) error {
		categories = append(categories, category)
		return nil
	})
	if err != nil {
		return err
	}
	return ds.indexer.AddPolicyCategories(categories)
}

// Search returns policy category related search results for the provided query
func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	if ok, err := policyCategorySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if ok, err := policyCategorySAC.ReadAllowed(ctx); err != nil || !ok {
		return 0, err
	}
	return ds.searcher.Count(ctx, q)
}

// GetPolicyCategory get a policy category by id
func (ds *datastoreImpl) GetPolicyCategory(ctx context.Context, id string) (*storage.PolicyCategory, bool, error) {
	if ok, err := policyCategorySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	category, exists, err := ds.storage.Get(ctx, id)
	if err != nil {
		return nil, false, errorsPkg.Wrapf(err, "policy category with id '%s' cannot be found	", id)
	}
	if !exists {
		return nil, false, errorsPkg.Wrapf(errox.NotFound, "policy category with id '%s' does not exist", id)
	}
	return category, true, nil
}

// SearchPolicyCategories returns search results that match the provided query
func (ds *datastoreImpl) SearchPolicyCategories(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchCategories(ctx, q)
}

// SearchRawPolicyCategories returns policy category objects that match the provided query
func (ds *datastoreImpl) SearchRawPolicyCategories(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategory, error) {
	if ok, err := policyCategorySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	return ds.searcher.SearchRawCategories(ctx, q)
}

// GetAllPolicyCategories lists all policy categories
func (ds *datastoreImpl) GetAllPolicyCategories(ctx context.Context) ([]*storage.PolicyCategory, error) {
	if ok, err := policyCategorySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	var categories []*storage.PolicyCategory
	err := ds.storage.Walk(ctx, func(category *storage.PolicyCategory) error {
		categories = append(categories, category)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return categories, err
}

// AddPolicyCategory inserts a policy category into the storage and the indexer
func (ds *datastoreImpl) AddPolicyCategory(ctx context.Context, category *storage.PolicyCategory) (*storage.PolicyCategory, error) {
	if ok, err := policyCategorySAC.WriteAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}
	if category.Id == "" {
		category.Id = uuid.NewV4().String()
	}
	// Any category added after startup must be marked custom category.
	category.IsDefault = false

	ds.categoryMutex.Lock()
	defer ds.categoryMutex.Unlock()

	category.Name = strings.Title(category.GetName())
	err := ds.storage.Upsert(ctx, category)
	if err != nil {
		return nil, err
	}
	return category, ds.indexer.AddPolicyCategory(category)
}

// RenamePolicyCategory renames a policy category
func (ds *datastoreImpl) RenamePolicyCategory(ctx context.Context, id, newName string) (*storage.PolicyCategory, error) {
	if ok, err := policyCategorySAC.WriteAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	ds.categoryMutex.Lock()
	defer ds.categoryMutex.Unlock()

	category, exists, err := ds.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errorsPkg.Wrapf(errox.NotFound, "policy category with id '%s' does not exist", id)
	}

	if category.GetIsDefault() {
		return nil, errorsPkg.Wrap(errox.InvalidArgs, fmt.Sprintf("policy category %q is a default category, cannot be renamed", id))
	}

	category.Name = strings.Title(newName)
	err = ds.storage.Upsert(ctx, category)
	if err != nil {
		return nil, errorsPkg.Wrap(err, fmt.Sprintf("failed to rename category '%q' to '%q'", id, newName))
	}

	err = ds.indexer.AddPolicyCategory(category)
	if err != nil {
		return nil, errorsPkg.Wrap(err, fmt.Sprintf("failed to rename category '%q' to '%q'", id, newName))
	}

	return &storage.PolicyCategory{
		Id:        category.GetId(),
		Name:      category.GetName(),
		IsDefault: category.GetIsDefault(),
	}, nil
}

// DeletePolicyCategory removes a policy from the storage and the indexer
func (ds *datastoreImpl) DeletePolicyCategory(ctx context.Context, id string) error {
	if ok, err := policyCategorySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.categoryMutex.Lock()
	defer ds.categoryMutex.Unlock()

	category, exists, err := ds.storage.Get(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if category.GetIsDefault() {
		return errorsPkg.Wrap(errox.InvalidArgs, fmt.Sprintf("policy category %q is a default category, cannot be removed", id))
	}
	//TODO: Wire in policy datastore, to search for policies using the category. If in use, category will not be deleted
	if err := ds.storage.Delete(ctx, id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicyCategory(id)
}
