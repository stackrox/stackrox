package datastore

import (
	"context"

	"github.com/stackrox/rox/central/policycategory/index"
	"github.com/stackrox/rox/central/policycategory/search"
	"github.com/stackrox/rox/central/policycategory/store"
	categoryUtils "github.com/stackrox/rox/central/policycategory/utils"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

// DataStore is an intermediary to policy category storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRawPolicyCategories(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategory, error)
	SearchPolicyCategories(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)

	GetPolicyCategory(ctx context.Context, id string) (*storage.PolicyCategory, bool, error)
	GetAllPolicyCategories(ctx context.Context) ([]*storage.PolicyCategory, error)
	GetPolicyCategoriesForPolicy(ctx context.Context, policyID string) ([]*storage.PolicyCategory, error)
	SetPolicyCategoriesForPolicy(ctx context.Context, policyID string, categories []string) error

	AddPolicyCategory(context.Context, *storage.PolicyCategory) (*storage.PolicyCategory, error)
	RenamePolicyCategory(ctx context.Context, id, newName string) (*storage.PolicyCategory, error)
	DeletePolicyCategory(ctx context.Context, id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(store store.Store, indexer index.Indexer, searcher search.Searcher, policyCategoryEdgeDS policyCategoryEdgeDS.DataStore) DataStore {
	ds := &datastoreImpl{
		storage:              store,
		indexer:              indexer,
		searcher:             searcher,
		policyCategoryEdgeDS: policyCategoryEdgeDS,
		categoryNameIDMap:    make(map[string]string, 0),
	}

	// set associations to provided categories
	categories := make([]*storage.PolicyCategory, 0)
	err := store.Walk(policyCategoryCtx, func(category *storage.PolicyCategory) error {
		categories = append(categories, category)
		return nil
	})
	if err != nil {
		utils.CrashOnError(err)
	}
	ds.categoryNameIDMap = categoryUtils.GetCategoryNameToIDs(categories)

	if err := ds.buildIndex(); err != nil {
		panic("unable to load search index for policy categories")
	}
	return ds
}

// newWithoutDefaults should be used only for testing purposes.
func newWithoutDefaults(store store.Store, indexer index.Indexer, searcher search.Searcher, policyCategoryEdgeDS policyCategoryEdgeDS.DataStore) DataStore {
	return &datastoreImpl{
		storage:              store,
		indexer:              indexer,
		searcher:             searcher,
		policyCategoryEdgeDS: policyCategoryEdgeDS,
		categoryNameIDMap:    make(map[string]string, 0),
	}
}
