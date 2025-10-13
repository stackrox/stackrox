//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/activecomponent/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestActiveComponentDatastore(t *testing.T) {
	suite.Run(t, new(ActiveComponentDatastoreSuite))
}

type ActiveComponentDatastoreSuite struct {
	suite.Suite

	hasReadCtx     context.Context
	hasWriteCtx    context.Context
	hasNoAccessCtx context.Context

	datastore DataStore
	testDB    *pgtest.TestPostgres
}

func (suite *ActiveComponentDatastoreSuite) SetupSuite() {
	suite.testDB = pgtest.ForT(suite.T())
	suite.Require().NotNil(suite.testDB)

	store := pgStore.New(suite.testDB.DB)
	suite.datastore = New(store)

	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
		),
	)

	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
		),
	)

	suite.hasNoAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
}

func (suite *ActiveComponentDatastoreSuite) SetupTest() {
	// Clean up the database before each test
	tag, err := suite.testDB.Exec(suite.hasWriteCtx, "TRUNCATE active_components CASCADE")
	suite.T().Log("active_components", tag)
	suite.NoError(err)
}

func (suite *ActiveComponentDatastoreSuite) TearDownSuite() {
	suite.testDB.Close()
}

func (suite *ActiveComponentDatastoreSuite) TestSearch() {
	// Insert a test component first
	component := suite.createTestActiveComponent()
	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component})
	suite.NoError(err)

	// Search for it
	results, err := suite.datastore.Search(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal(component.GetId(), results[0].ID)
}

func (suite *ActiveComponentDatastoreSuite) TestSearchNoAccess() {
	// Insert a test component first
	component := suite.createTestActiveComponent()
	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component})
	suite.NoError(err)

	// Search with no access should return empty results
	results, err := suite.datastore.Search(suite.hasNoAccessCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *ActiveComponentDatastoreSuite) TestSearchRawActiveComponents() {
	// Insert test components
	component1 := suite.createTestActiveComponent()
	component2 := suite.createTestActiveComponent()
	components := []*storage.ActiveComponent{component1, component2}

	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, components)
	suite.NoError(err)

	// Search for them
	foundComponents, err := suite.datastore.SearchRawActiveComponents(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Len(foundComponents, 2)

	// Verify the components match
	foundIDs := make([]string, len(foundComponents))
	for i, comp := range foundComponents {
		foundIDs[i] = comp.GetId()
	}
	suite.Contains(foundIDs, component1.GetId())
	suite.Contains(foundIDs, component2.GetId())
}

func (suite *ActiveComponentDatastoreSuite) TestGet() {
	// Insert a test component
	component := suite.createTestActiveComponent()
	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component})
	suite.NoError(err)

	// Get the component
	foundComponent, found, err := suite.datastore.Get(suite.hasReadCtx, component.GetId())
	suite.NoError(err)
	suite.True(found)
	protoassert.Equal(suite.T(), component, foundComponent)
}

func (suite *ActiveComponentDatastoreSuite) TestGetNotFound() {
	id := "nonexistent-id"
	component, found, err := suite.datastore.Get(suite.hasReadCtx, id)
	suite.NoError(err)
	suite.False(found)
	suite.Nil(component)
}

func (suite *ActiveComponentDatastoreSuite) TestGetNoAccess() {
	// Insert a test component
	component := suite.createTestActiveComponent()
	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component})
	suite.NoError(err)

	// Try to get with no access
	foundComponent, found, err := suite.datastore.Get(suite.hasNoAccessCtx, component.GetId())
	suite.NoError(err)
	suite.False(found)
	suite.Nil(foundComponent)
}

func (suite *ActiveComponentDatastoreSuite) TestExists() {
	// Insert a test component
	component := suite.createTestActiveComponent()
	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component})
	suite.NoError(err)

	// Check if it exists
	exists, err := suite.datastore.Exists(suite.hasReadCtx, component.GetId())
	suite.NoError(err)
	suite.True(exists)
}

func (suite *ActiveComponentDatastoreSuite) TestExistsNotFound() {
	id := "nonexistent-id"
	exists, err := suite.datastore.Exists(suite.hasReadCtx, id)
	suite.NoError(err)
	suite.False(exists)
}

func (suite *ActiveComponentDatastoreSuite) TestGetBatch() {
	// Insert test components
	component1 := suite.createTestActiveComponent()
	component2 := suite.createTestActiveComponent()
	component3 := suite.createTestActiveComponent()

	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component1, component2})
	suite.NoError(err)

	// Get batch including one non-existent component
	ids := []string{component1.GetId(), component2.GetId(), component3.GetId()}
	components, err := suite.datastore.GetBatch(suite.hasReadCtx, ids)
	suite.NoError(err)
	suite.Len(components, 2) // Only 2 should be found

	foundIDs := make([]string, len(components))
	for i, comp := range components {
		foundIDs[i] = comp.GetId()
	}
	suite.Contains(foundIDs, component1.GetId())
	suite.Contains(foundIDs, component2.GetId())
}

func (suite *ActiveComponentDatastoreSuite) TestUpsertBatch() {
	components := []*storage.ActiveComponent{
		suite.createTestActiveComponent(),
		suite.createTestActiveComponent(),
	}

	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, components)
	suite.NoError(err)

	// Verify they were inserted
	for _, component := range components {
		foundComponent, found, err := suite.datastore.Get(suite.hasReadCtx, component.GetId())
		suite.NoError(err)
		suite.True(found)
		protoassert.Equal(suite.T(), component, foundComponent)
	}
}

func (suite *ActiveComponentDatastoreSuite) TestUpsertBatchNoAccess() {
	components := []*storage.ActiveComponent{
		suite.createTestActiveComponent(),
	}

	err := suite.datastore.UpsertBatch(suite.hasNoAccessCtx, components)
	suite.Error(err)
	suite.Equal(sac.ErrResourceAccessDenied, err)
}

func (suite *ActiveComponentDatastoreSuite) TestUpsertBatchUpdate() {
	// Insert a component
	component := suite.createTestActiveComponent()
	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component})
	suite.NoError(err)

	// Update it
	component.ComponentId = "updated-component-id"
	err = suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component})
	suite.NoError(err)

	// Verify the update
	foundComponent, found, err := suite.datastore.Get(suite.hasReadCtx, component.GetId())
	suite.NoError(err)
	suite.True(found)
	suite.Equal("updated-component-id", foundComponent.GetComponentId())
}

func (suite *ActiveComponentDatastoreSuite) TestDeleteBatch() {
	// Insert test components
	component1 := suite.createTestActiveComponent()
	component2 := suite.createTestActiveComponent()
	component3 := suite.createTestActiveComponent()

	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component1, component2, component3})
	suite.NoError(err)

	// Delete two of them
	ids := []string{component1.GetId(), component2.GetId()}
	err = suite.datastore.DeleteBatch(suite.hasWriteCtx, ids...)
	suite.NoError(err)

	// Verify they were deleted
	_, found, err := suite.datastore.Get(suite.hasReadCtx, component1.GetId())
	suite.NoError(err)
	suite.False(found)

	_, found, err = suite.datastore.Get(suite.hasReadCtx, component2.GetId())
	suite.NoError(err)
	suite.False(found)

	// Verify the third one still exists
	_, found, err = suite.datastore.Get(suite.hasReadCtx, component3.GetId())
	suite.NoError(err)
	suite.True(found)
}

func (suite *ActiveComponentDatastoreSuite) TestDeleteBatchNoAccess() {
	// Insert a test component
	component := suite.createTestActiveComponent()
	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{component})
	suite.NoError(err)

	// Try to delete with no access
	err = suite.datastore.DeleteBatch(suite.hasNoAccessCtx, component.GetId())
	suite.Error(err)
	suite.Equal(sac.ErrResourceAccessDenied, err)

	// Verify it still exists
	_, found, err := suite.datastore.Get(suite.hasReadCtx, component.GetId())
	suite.NoError(err)
	suite.True(found)
}

func (suite *ActiveComponentDatastoreSuite) TestDeleteBatchNonExistent() {
	// Try to delete non-existent components - should not error
	ids := []string{"nonexistent-1", "nonexistent-2"}
	err := suite.datastore.DeleteBatch(suite.hasWriteCtx, ids...)
	suite.NoError(err)
}

func (suite *ActiveComponentDatastoreSuite) TestDeleteBatchEmptyIDs() {
	// Delete with empty IDs should not error
	var ids []string
	err := suite.datastore.DeleteBatch(suite.hasWriteCtx, ids...)
	suite.NoError(err)
}

// Helper function to create test active components
func (suite *ActiveComponentDatastoreSuite) createTestActiveComponent() *storage.ActiveComponent {
	return &storage.ActiveComponent{
		Id:           uuid.NewV4().String(),
		DeploymentId: uuid.NewV4().String(),
		ComponentId:  uuid.NewV4().String(),
		ActiveContextsSlice: []*storage.ActiveComponent_ActiveContext{
			{
				ContainerName: "test-container",
				ImageId:       "test-image-id",
			},
		},
	}
}

func (suite *ActiveComponentDatastoreSuite) TestComplexWorkflow() {
	// Test a complex workflow with multiple operations
	components := []*storage.ActiveComponent{
		suite.createTestActiveComponent(),
		suite.createTestActiveComponent(),
		suite.createTestActiveComponent(),
	}

	// Insert all components
	err := suite.datastore.UpsertBatch(suite.hasWriteCtx, components)
	suite.NoError(err)

	// Search and verify all are present
	results, err := suite.datastore.Search(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 3)

	// Update one component
	components[0].ComponentId = "updated-component"
	err = suite.datastore.UpsertBatch(suite.hasWriteCtx, []*storage.ActiveComponent{components[0]})
	suite.NoError(err)

	// Verify the update
	foundComponent, found, err := suite.datastore.Get(suite.hasReadCtx, components[0].GetId())
	suite.NoError(err)
	suite.True(found)
	suite.Equal("updated-component", foundComponent.GetComponentId())

	// Delete two components
	err = suite.datastore.DeleteBatch(suite.hasWriteCtx, components[0].GetId(), components[1].GetId())
	suite.NoError(err)

	// Verify only one component remains
	results, err = suite.datastore.Search(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal(components[2].GetId(), results[0].ID)
}
