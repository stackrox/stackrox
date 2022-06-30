package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	indexMocks "github.com/stackrox/rox/central/policycategory/index/mocks"
	storeMocks "github.com/stackrox/rox/central/policycategory/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestPolicyCategoryDatastore(t *testing.T) {
	suite.Run(t, new(PolicyCategoryDatastoreTestSuite))
}

type PolicyCategoryDatastoreTestSuite struct {
	suite.Suite

	mockCtrl  *gomock.Controller
	store     *storeMocks.MockStore
	indexer   *indexMocks.MockIndexer
	datastore DataStore

	ctx context.Context
}

func (s *PolicyCategoryDatastoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)

	s.datastore = newWithoutDefaults(s.store, s.indexer, nil)

	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
}

func (s *PolicyCategoryDatastoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PolicyCategoryDatastoreTestSuite) TestAddNewPolicyCategory() {
	s.store.EXPECT().Upsert(s.ctx, gomock.Any()).Return(nil)
	s.store.EXPECT().Walk(s.ctx, gomock.Any())

	s.indexer.EXPECT().AddPolicyCategory(gomock.Any()).Return(nil)

	_, err := s.datastore.AddPolicyCategory(s.ctx, fixtures.GetPolicyCategory())
	s.NoError(err, "expected no error trying to add a category with permissions")
}

func (s *PolicyCategoryDatastoreTestSuite) TestAddDuplicatePolicyCategory() {
	mockCategories := []*storage.PolicyCategory{
		{
			Id:        "boo-1",
			Name:      "Boo's Special Category",
			IsDefault: true,
		},
	}

	s.store.EXPECT().Walk(s.ctx, gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(c *storage.PolicyCategory) error) error {
			for _, c := range mockCategories {
				_ = fn(c)
			}
			return nil
		})

	_, err := s.datastore.AddPolicyCategory(s.ctx, fixtures.GetPolicyCategory())
	s.Error(err)

}

func (s *PolicyCategoryDatastoreTestSuite) TestDeletePolicyCategory() {
	c := fixtures.GetPolicyCategory()

	s.store.EXPECT().Upsert(s.ctx, gomock.Any()).Return(nil)
	s.store.EXPECT().Delete(s.ctx, gomock.Any()).Return(nil).AnyTimes()
	s.store.EXPECT().Walk(s.ctx, gomock.Any()).AnyTimes()

	s.indexer.EXPECT().AddPolicyCategory(gomock.Any()).Return(nil).AnyTimes()
	s.indexer.EXPECT().DeletePolicyCategory(gomock.Any()).Return(nil).AnyTimes()

	_, err := s.datastore.AddPolicyCategory(s.ctx, c)
	s.NoError(err)

	err = s.datastore.DeletePolicyCategory(s.ctx, c.Id)
	s.NoError(err, "expected no error trying to delete a category with permissions")
}

func (s *PolicyCategoryDatastoreTestSuite) TestRenamePolicyCategory() {
	c := fixtures.GetPolicyCategory()

	s.store.EXPECT().Upsert(s.ctx, gomock.Any()).Return(nil).AnyTimes()
	s.store.EXPECT().Walk(s.ctx, gomock.Any()).AnyTimes()
	s.store.EXPECT().Get(s.ctx, c.GetId()).Return(c, true, nil)

	s.indexer.EXPECT().AddPolicyCategory(gomock.Any()).Return(nil).AnyTimes()

	_, err := s.datastore.AddPolicyCategory(s.ctx, c)
	s.NoError(err)

	err = s.datastore.RenamePolicyCategory(s.ctx, c.Id, "Boo's Special Category New Name")
	s.NoError(err, "expected no error trying to rename a category with permissions")
}

func (s *PolicyCategoryDatastoreTestSuite) TestRenamePolicyCategoryDuplicateName() {
	addedCategories := []*storage.PolicyCategory{
		fixtures.GetPolicyCategory(),
		{
			Id:        "new-id",
			Name:      "New Name",
			IsDefault: false,
		},
	}
	s.store.EXPECT().Upsert(s.ctx, gomock.Any()).Return(nil).AnyTimes()
	s.store.EXPECT().Walk(s.ctx, gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, fn func(c *storage.PolicyCategory) error) error {
			for _, c := range addedCategories {
				_ = fn(c)
			}
			return nil
		})
	s.store.EXPECT().Get(s.ctx, "category-id").Return(fixtures.GetPolicyCategory(), true, nil)

	err := s.datastore.RenamePolicyCategory(s.ctx, "category-id", "new name")
	s.Error(err, "expected an error trying to rename a category with permissions, but with a duplicate name")
}
