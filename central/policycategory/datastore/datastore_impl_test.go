package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	indexMocks "github.com/stackrox/rox/central/policycategory/index/mocks"
	storeMocks "github.com/stackrox/rox/central/policycategory/store/mocks"
	policyCategoryEdgeDSMocks "github.com/stackrox/rox/central/policycategoryedge/datastore/mocks"
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

	mockCtrl      *gomock.Controller
	store         *storeMocks.MockStore
	indexer       *indexMocks.MockIndexer
	edgeDataStore *policyCategoryEdgeDSMocks.MockDataStore
	datastore     DataStore

	hasReadWritePolicyAccess                 context.Context
	hasReadWriteWorkflowAdministrationAccess context.Context
}

func (s *PolicyCategoryDatastoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)
	s.edgeDataStore = policyCategoryEdgeDSMocks.NewMockDataStore(s.mockCtrl)

	s.datastore = newWithoutDefaults(s.store, s.indexer, nil, s.edgeDataStore)

	s.hasReadWritePolicyAccess = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
	// TODO: ROX-13888 Remove duplicated context.
	s.hasReadWriteWorkflowAdministrationAccess = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
}

func (s *PolicyCategoryDatastoreTestSuite) TestAddNewPolicyCategory() {
	s.store.EXPECT().Upsert(s.hasReadWritePolicyAccess, gomock.Any()).Return(nil).Times(1)

	s.indexer.EXPECT().AddPolicyCategory(gomock.Any()).Return(nil).AnyTimes()

	_, err := s.datastore.AddPolicyCategory(s.hasReadWritePolicyAccess, fixtures.GetPolicyCategory())
	s.NoError(err, "expected no error trying to add a category with permissions")
	// TODO: ROX-13888 Remove duplicated test.
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil).Times(1)
	_, err = s.datastore.AddPolicyCategory(s.hasReadWriteWorkflowAdministrationAccess, fixtures.GetPolicyCategory())
	s.NoError(err, "expected no error trying to add a category with permissions")
}

func (s *PolicyCategoryDatastoreTestSuite) TestAddDuplicatePolicyCategory() {
	c := fixtures.GetPolicyCategory()
	s.store.EXPECT().Upsert(s.hasReadWritePolicyAccess, c).Return(errors.New("exists"))

	_, err := s.datastore.AddPolicyCategory(s.hasReadWritePolicyAccess, c)
	s.Error(err)
}

func (s *PolicyCategoryDatastoreTestSuite) TestDeletePolicyCategory() {
	c := fixtures.GetPolicyCategory()

	s.store.EXPECT().Get(s.hasReadWritePolicyAccess, "category-id").Return(c, true, nil).Times(1)
	s.store.EXPECT().Delete(s.hasReadWritePolicyAccess, gomock.Any()).Return(nil).AnyTimes()

	s.indexer.EXPECT().DeletePolicyCategory(gomock.Any()).Return(nil).AnyTimes()

	err := s.datastore.DeletePolicyCategory(s.hasReadWritePolicyAccess, c.Id)
	s.NoError(err, "expected no error trying to delete a category with permissions")
	// TODO: ROX-13888 Remove duplicated test.
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, "category-id").Return(c, true, nil).Times(1)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil).AnyTimes()
	err = s.datastore.DeletePolicyCategory(s.hasReadWriteWorkflowAdministrationAccess, c.Id)
	s.NoError(err, "expected no error trying to delete a category with permissions")
}

func (s *PolicyCategoryDatastoreTestSuite) TestDeleteDefaultPolicyCategory() {
	c := fixtures.GetPolicyCategory()
	c.IsDefault = true

	s.store.EXPECT().Get(s.hasReadWritePolicyAccess, "category-id").Return(c, true, nil)

	err := s.datastore.DeletePolicyCategory(s.hasReadWritePolicyAccess, c.Id)
	s.Error(err)
}

func (s *PolicyCategoryDatastoreTestSuite) TestRenamePolicyCategory() {
	c := fixtures.GetPolicyCategory()

	s.store.EXPECT().Upsert(s.hasReadWritePolicyAccess, gomock.Any()).Return(nil).AnyTimes()
	s.store.EXPECT().Get(s.hasReadWritePolicyAccess, c.GetId()).Return(c, true, nil)

	s.indexer.EXPECT().AddPolicyCategory(gomock.Any()).Return(nil).AnyTimes()

	c, err := s.datastore.RenamePolicyCategory(s.hasReadWritePolicyAccess, c.Id, "Boo's Special Category New Name")
	s.NoError(err, "expected no error trying to rename a category with permissions")
	s.Equal("Boo'S Special Category New Name", c.GetName(), "expected category to be renamed, but it is not")

	// TODO: ROX-13888 Remove duplicated test.
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil).AnyTimes()
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, c.GetId()).Return(c, true, nil)
	c, err = s.datastore.RenamePolicyCategory(s.hasReadWriteWorkflowAdministrationAccess, c.Id, "Daniel's Special Category New Name")
	s.NoError(err, "expected no error trying to rename a category with permissions")
	s.Equal("Daniel'S Special Category New Name", c.GetName(), "expected category to be renamed, but it is not")
}

func (s *PolicyCategoryDatastoreTestSuite) TestRenamePolicyCategoryDuplicateName() {
	s.store.EXPECT().Upsert(s.hasReadWritePolicyAccess, gomock.Any()).Return(errors.New("exists")).AnyTimes()
	s.store.EXPECT().Get(s.hasReadWritePolicyAccess, "category-id").Return(fixtures.GetPolicyCategory(), true, nil)

	_, err := s.datastore.RenamePolicyCategory(s.hasReadWritePolicyAccess, "category-id", "new name")
	s.Error(err)
}

func (s *PolicyCategoryDatastoreTestSuite) TestRenameDefaultPolicyCategory() {
	c := fixtures.GetPolicyCategory()
	c.IsDefault = true

	s.store.EXPECT().Get(s.hasReadWritePolicyAccess, "category-id").Return(c, true, nil)

	_, err := s.datastore.RenamePolicyCategory(s.hasReadWritePolicyAccess, c.Id, "new name")
	s.Error(err)
}
