package datastore

import (
	"context"
	"fmt"
	"strings"

	errorsPkg "github.com/pkg/errors"
	"github.com/stackrox/rox/central/policycategory/search"
	"github.com/stackrox/rox/central/policycategory/store"
	"github.com/stackrox/rox/central/policycategoryedge/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log               = logging.LoggerForModule()
	policyCategorySAC = sac.ForResource(resources.WorkflowAdministration)

	policyCategoryCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
)

type datastoreImpl struct {
	storage              store.Store
	searcher             search.Searcher
	policyCategoryEdgeDS datastore.DataStore
	categoryMutex        sync.Mutex

	categoryNameIDMap map[string]string
}

func (ds *datastoreImpl) SetPolicyCategoriesForPolicy(ctx context.Context, policyID string, categoryNames []string) error {
	ds.categoryMutex.Lock()
	defer ds.categoryMutex.Unlock()

	edges, err := ds.policyCategoryEdgeDS.SearchRawEdges(ctx, searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.PolicyID, policyID).ProtoQuery())
	if err != nil {
		return err
	}
	existingCategoryIDs := make([]string, 0, len(edges))
	for _, e := range edges {
		existingCategoryIDs = append(existingCategoryIDs, e.GetCategoryId())
	}
	existingCategories, _, err := ds.storage.GetMany(ctx, existingCategoryIDs)
	if err != nil {
		return err
	}
	existingCategoryNames := set.NewStringSet()
	for _, c := range existingCategories {
		existingCategoryNames.Add(c.GetName())
	}

	categoriesToUpdate := set.NewStringSet(categoryNames...)
	if len(existingCategoryNames.Difference(categoriesToUpdate).AsSlice()) == 0 &&
		len(categoriesToUpdate.Difference(existingCategoryNames).AsSlice()) == 0 {
		// no edges to update since existing categories and categories to be updated are the same
		return nil
	}

	// disassociate all categories from given policy
	if err := ds.policyCategoryEdgeDS.DeleteByQuery(ctx, searchPkg.NewQueryBuilder().AddExactMatches(searchPkg.PolicyID, policyID).ProtoQuery()); err != nil {
		return err
	}
	if len(categoryNames) == 0 {
		return nil
	}

	categoryIds := make([]string, 0, len(categoryNames))
	categoriesToAdd := make([]*storage.PolicyCategory, 0)
	for _, c := range categoryNames {
		if ds.categoryNameIDMap[strings.Title(c)] != "" {
			categoryIds = append(categoryIds, ds.categoryNameIDMap[c])
		} else {
			newCategory := &storage.PolicyCategory{
				Id:        uuid.NewV4().String(),
				Name:      strings.Title(c),
				IsDefault: false,
			}
			categoriesToAdd = append(categoriesToAdd, newCategory)
			categoryIds = append(categoryIds, newCategory.GetId())
		}
	}

	err = ds.storage.UpsertMany(ctx, categoriesToAdd)
	if err != nil {
		return err
	}

	for _, c := range categoriesToAdd {
		ds.categoryNameIDMap[c.GetName()] = c.GetId()
	}

	policyCategoryEdges := make([]*storage.PolicyCategoryEdge, 0, len(categoryIds))
	for _, id := range categoryIds {
		policyCategoryEdges = append(policyCategoryEdges, &storage.PolicyCategoryEdge{
			Id:         uuid.NewV4().String(),
			PolicyId:   policyID,
			CategoryId: id,
		})
	}
	return ds.policyCategoryEdgeDS.UpsertMany(ctx, policyCategoryEdges)
}

func (ds *datastoreImpl) GetPolicyCategoriesForPolicy(ctx context.Context, policyID string) ([]*storage.PolicyCategory, error) {
	ds.categoryMutex.Lock()
	defer ds.categoryMutex.Unlock()

	return ds.SearchRawPolicyCategories(ctx, searchPkg.NewQueryBuilder().AddStrings(searchPkg.PolicyID, policyID).ProtoQuery())
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
	walkFn := func() error {
		categories = categories[:0]
		return ds.storage.Walk(ctx, func(category *storage.PolicyCategory) error {
			categories = append(categories, category)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return categories, nil
}

// AddPolicyCategory inserts a policy category into the storage.
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
	ds.categoryNameIDMap[category.GetName()] = category.GetId()

	return category, nil
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
	existingCategoryName := category.GetName()

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
	delete(ds.categoryNameIDMap, existingCategoryName)
	ds.categoryNameIDMap[category.GetName()] = category.GetId()

	return &storage.PolicyCategory{
		Id:        category.GetId(),
		Name:      category.GetName(),
		IsDefault: category.GetIsDefault(),
	}, nil
}

// DeletePolicyCategory removes a policy from the storage
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

	if err := ds.storage.Delete(ctx, id); err != nil {
		return err
	}
	delete(ds.categoryNameIDMap, category.GetName())
	return nil
}
