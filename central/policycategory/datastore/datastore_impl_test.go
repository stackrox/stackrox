package datastore

import (
	"context"
	"errors"
	"testing"

	indexMocks "github.com/stackrox/rox/central/policycategory/index/mocks"
	storeMocks "github.com/stackrox/rox/central/policycategory/store/mocks"
	policyCategoryEdgeDSMocks "github.com/stackrox/rox/central/policycategoryedge/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPolicyCategoryDatastore(t *testing.T) {
	suite.Run(t, new(PolicyCategoryDatastoreTestSuite))
}

type PolicyCategoryDatastoreTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	store         *storeMocks.MockStore
	indexer       *indexMocks.MockIndexer
	edgeDataStore *policyCategoryEdgeDSMocks.MockDataStore
	datastore     DataStore

	hasReadWriteWorkflowAdministrationCtx context.Context
}

func (s *PolicyCategoryDatastoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)
	s.edgeDataStore = policyCategoryEdgeDSMocks.NewMockDataStore(s.mockCtrl)

	s.datastore = newWithoutDefaults(s.store, nil, s.edgeDataStore)

	s.hasReadWriteWorkflowAdministrationCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (s *PolicyCategoryDatastoreTestSuite) TestAddNewPolicyCategory() {
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationCtx, gomock.Any()).Return(nil).Times(1)
	_, err := s.datastore.AddPolicyCategory(s.hasReadWriteWorkflowAdministrationCtx, fixtures.GetPolicyCategory())
	s.NoError(err, "expected no error trying to add a category with permissions")
}

func (s *PolicyCategoryDatastoreTestSuite) TestAddDuplicatePolicyCategory() {
	c := fixtures.GetPolicyCategory()
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationCtx, c).Return(errors.New("exists"))

	_, err := s.datastore.AddPolicyCategory(s.hasReadWriteWorkflowAdministrationCtx, c)
	s.Error(err)
}

func (s *PolicyCategoryDatastoreTestSuite) TestDeletePolicyCategory() {
	c := fixtures.GetPolicyCategory()

	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationCtx, "category-id").Return(c, true, nil).Times(1)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationCtx, gomock.Any()).Return(nil).AnyTimes()
	err := s.datastore.DeletePolicyCategory(s.hasReadWriteWorkflowAdministrationCtx, c.Id)
	s.NoError(err, "expected no error trying to delete a category with permissions")
}

func (s *PolicyCategoryDatastoreTestSuite) TestDeleteDefaultPolicyCategory() {
	c := fixtures.GetPolicyCategory()
	c.IsDefault = true

	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationCtx, "category-id").Return(c, true, nil)

	err := s.datastore.DeletePolicyCategory(s.hasReadWriteWorkflowAdministrationCtx, c.Id)
	s.Error(err)
}

func (s *PolicyCategoryDatastoreTestSuite) TestRenamePolicyCategory() {
	c := fixtures.GetPolicyCategory()

	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationCtx, gomock.Any()).Return(nil).AnyTimes()
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationCtx, c.GetId()).Return(c, true, nil)
	c, err := s.datastore.RenamePolicyCategory(s.hasReadWriteWorkflowAdministrationCtx, c.Id, "Boo's Special Category New Name")
	s.NoError(err, "expected no error trying to rename a category with permissions")
	s.Equal("Boo'S Special Category New Name", c.GetName(), "expected category to be renamed, but it is not")
}

func (s *PolicyCategoryDatastoreTestSuite) TestRenamePolicyCategoryDuplicateName() {
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationCtx, gomock.Any()).Return(errors.New("exists")).AnyTimes()
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationCtx, "category-id").Return(fixtures.GetPolicyCategory(), true, nil)

	_, err := s.datastore.RenamePolicyCategory(s.hasReadWriteWorkflowAdministrationCtx, "category-id", "new name")
	s.Error(err)
}

func (s *PolicyCategoryDatastoreTestSuite) TestRenameDefaultPolicyCategory() {
	c := fixtures.GetPolicyCategory()
	c.IsDefault = true

	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationCtx, "category-id").Return(c, true, nil)

	_, err := s.datastore.RenamePolicyCategory(s.hasReadWriteWorkflowAdministrationCtx, c.Id, "new name")
	s.Error(err)
}
