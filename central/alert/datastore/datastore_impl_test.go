//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	matcherMocks "github.com/stackrox/rox/central/platform/matcher/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

type AlertDatastoreImplSuite struct {
	suite.Suite

	testPostgres *pgtest.TestPostgres
	datastore    DataStore
	matcher      *matcherMocks.MockPlatformMatcher
	mockCtrl     *gomock.Controller

	// Track alert IDs created during tests for cleanup
	createdAlertIDs []string
}

func TestAlertDatastoreImpl(t *testing.T) {
	suite.Run(t, new(AlertDatastoreImplSuite))
}

func (s *AlertDatastoreImplSuite) SetupTest() {
	s.testPostgres = pgtest.ForT(s.T())
	s.mockCtrl = gomock.NewController(s.T())
	s.matcher = matcherMocks.NewMockPlatformMatcher(s.mockCtrl)

	store := postgres.New(s.testPostgres.DB)
	s.datastore = New(store, s.matcher)

	// Initialize alert tracking
	s.createdAlertIDs = []string{}
}

func (s *AlertDatastoreImplSuite) TearDownTest() {
	// Clean up any alerts created during the test
	if len(s.createdAlertIDs) > 0 {
		_ = s.datastore.DeleteAlerts(ctx, s.createdAlertIDs...)
	}
}

// Helper method to create an alert and track it for cleanup
func (s *AlertDatastoreImplSuite) createAndTrackAlert(alert *storage.Alert) {
	s.createdAlertIDs = append(s.createdAlertIDs, alert.GetId())
	err := s.datastore.UpsertAlert(ctx, alert)
	s.NoError(err)
}

// TestSearch covers the same functionality as searcher_postgres_test.go TestSearch
func (s *AlertDatastoreImplSuite) TestSearch() {
	alert := fixtures.GetAlert()
	alert.EntityType = storage.Alert_DEPLOYMENT
	alert.PlatformComponent = false

	// Mock platform matcher to return false for platform component
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)

	// Test alert doesn't exist initially
	foundAlert, exists, err := s.datastore.GetAlert(ctx, alert.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundAlert)

	// Upsert the alert
	s.NoError(s.datastore.UpsertAlert(ctx, alert))
	foundAlert, exists, err = s.datastore.GetAlert(ctx, alert.GetId())
	s.NoError(err)
	s.True(exists)
	protoassert.Equal(s.T(), alert, foundAlert)

	// Test common alert searches
	results, err := s.datastore.Search(ctx, search.NewQueryBuilder().AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).ProtoQuery(), true)
	s.NoError(err)
	s.Len(results, 1)

	q := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).
		AddExactMatches(search.PolicyID, alert.GetPolicy().GetId()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		ProtoQuery()
	results, err = s.datastore.Search(ctx, q, true)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().
		AddBools(search.PlatformComponent, false).
		AddExactMatches(search.EntityType, storage.Alert_DEPLOYMENT.String()).
		ProtoQuery()
	results, err = s.datastore.Search(ctx, q, true)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().
		AddBools(search.PlatformComponent, true).
		ProtoQuery()
	results, err = s.datastore.Search(ctx, q, true)
	s.NoError(err)
	s.Len(results, 0)
}

// TestSearchResolved covers the same functionality as searcher_postgres_test.go TestSearchResolved
func (s *AlertDatastoreImplSuite) TestSearchResolved() {
	ids := []string{fixtureconsts.Alert1, fixtureconsts.Alert2, fixtureconsts.Alert3, fixtureconsts.Alert4}
	allAlertIds := make(map[string]bool)
	unresolvedAlertIds := make(map[string]bool)

	// Mock platform matcher to return false for all alerts
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil).Times(len(ids))

	for i, id := range ids {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false
		if i >= 2 {
			alert.State = storage.ViolationState_RESOLVED
		} else {
			unresolvedAlertIds[alert.GetId()] = true
		}
		allAlertIds[alert.GetId()] = true
		s.NoError(s.datastore.UpsertAlert(ctx, alert))
		foundAlert, exists, err := s.datastore.GetAlert(ctx, id)
		s.True(exists)
		s.NoError(err)
		protoassert.Equal(s.T(), alert, foundAlert)
	}

	// Test search including resolved alerts
	results, err := s.datastore.Search(ctx, search.EmptyQuery(), false)
	s.NoError(err)
	// Check that all alerts are found and mark them as found
	for _, result := range results {
		s.True(allAlertIds[result.ID])
		allAlertIds[result.ID] = false
	}
	// Check that all ids were found
	for entry := range allAlertIds {
		s.False(allAlertIds[entry])
	}

	// Test search excluding resolved alerts (only unresolved)
	results, err = s.datastore.Search(ctx, search.EmptyQuery(), true)
	s.NoError(err)
	for _, result := range results {
		s.True(unresolvedAlertIds[result.ID])
		unresolvedAlertIds[result.ID] = false
	}
	for entry := range unresolvedAlertIds {
		s.False(unresolvedAlertIds[entry])
	}
}

// TestCountResolved covers the same functionality as searcher_postgres_test.go TestCountResolved
func (s *AlertDatastoreImplSuite) TestCountResolved() {
	ids := []string{fixtureconsts.Alert1, fixtureconsts.Alert2, fixtureconsts.Alert3, fixtureconsts.Alert4}

	// Mock platform matcher to return false for all alerts
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil).Times(len(ids))

	for i, id := range ids {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false
		if i >= 2 {
			alert.State = storage.ViolationState_RESOLVED
		}
		s.NoError(s.datastore.UpsertAlert(ctx, alert))
		foundAlert, exists, err := s.datastore.GetAlert(ctx, id)
		s.True(exists)
		s.NoError(err)
		protoassert.Equal(s.T(), alert, foundAlert)
	}

	// Test count including resolved alerts
	results, err := s.datastore.Count(ctx, search.EmptyQuery(), false)
	s.NoError(err)
	s.Equal(4, results)

	// Test count excluding resolved alerts (only unresolved)
	results, err = s.datastore.Count(ctx, search.EmptyQuery(), true)
	s.NoError(err)
	s.Equal(2, results)
}

// TestSearchAlerts tests the SearchAlerts functionality with real data
func (s *AlertDatastoreImplSuite) TestSearchAlerts() {
	alert := fixtures.GetAlert()
	alert.EntityType = storage.Alert_DEPLOYMENT
	alert.PlatformComponent = false

	// Mock platform matcher to return false for platform component
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)

	// Upsert the alert
	s.NoError(s.datastore.UpsertAlert(ctx, alert))

	// Test SearchAlerts
	searchResults, err := s.datastore.SearchAlerts(ctx, search.EmptyQuery())
	s.NoError(err)
	s.Len(searchResults, 1)
	s.Equal(alert.GetId(), searchResults[0].GetId())
	s.Equal(alert.GetPolicy().GetName(), searchResults[0].GetName())
}

// TestSearchRawAlerts tests the SearchRawAlerts functionality with real data
func (s *AlertDatastoreImplSuite) TestSearchRawAlerts() {
	alert := fixtures.GetAlert()
	alert.EntityType = storage.Alert_DEPLOYMENT
	alert.PlatformComponent = false

	// Mock platform matcher to return false for platform component
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)

	// Upsert the alert
	s.NoError(s.datastore.UpsertAlert(ctx, alert))

	// Test SearchRawAlerts excluding resolved
	rawAlerts, err := s.datastore.SearchRawAlerts(ctx, search.EmptyQuery(), true)
	s.NoError(err)
	s.Len(rawAlerts, 1)
	protoassert.Equal(s.T(), alert, rawAlerts[0])

	// Mark alert as resolved
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
	resolvedAlerts, err := s.datastore.MarkAlertsResolvedBatch(ctx, alert.GetId())
	s.NoError(err)
	s.Len(resolvedAlerts, 1)
	s.Equal(storage.ViolationState_RESOLVED, resolvedAlerts[0].GetState())

	// Test SearchRawAlerts excluding resolved (should return 0)
	rawAlerts, err = s.datastore.SearchRawAlerts(ctx, search.EmptyQuery(), true)
	s.NoError(err)
	s.Len(rawAlerts, 0)

	// Test SearchRawAlerts including resolved (should return 1)
	rawAlerts, err = s.datastore.SearchRawAlerts(ctx, search.EmptyQuery(), false)
	s.NoError(err)
	s.Len(rawAlerts, 1)
	s.Equal(storage.ViolationState_RESOLVED, rawAlerts[0].GetState())
}

// TestSearchListAlerts tests the SearchListAlerts functionality with pagination
func (s *AlertDatastoreImplSuite) TestSearchListAlerts() {
	// Create multiple alerts to test pagination
	alertIDs := []string{
		fixtureconsts.Deployment1,
		fixtureconsts.Deployment2,
		fixtureconsts.Deployment3,
		fixtureconsts.Deployment4,
		fixtureconsts.Deployment5,
	}

	var createdAlerts []*storage.Alert
	for i, id := range alertIDs {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false
		// Vary the policy names to make alerts distinguishable
		alert.Policy.Name = fmt.Sprintf("Test Policy %d", i+1)

		s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
		s.createAndTrackAlert(alert)
		createdAlerts = append(createdAlerts, alert)
	}

	// Test 1: Search without pagination (should return all alerts)
	listAlerts, err := s.datastore.SearchListAlerts(ctx, search.EmptyQuery(), true)
	s.NoError(err)
	s.Len(listAlerts, len(alertIDs))

	// Test 2: Search with limit (first 2 alerts)
	queryWithLimit := &v1.Query{
		Pagination: &v1.QueryPagination{
			Limit: 2,
		},
	}
	listAlerts, err = s.datastore.SearchListAlerts(ctx, queryWithLimit, true)
	s.NoError(err)
	s.Len(listAlerts, 2)

	// Verify the returned alerts are from our created set
	returnedIDs := make(map[string]bool)
	for _, alert := range listAlerts {
		returnedIDs[alert.GetId()] = true
		// Ensure it's one of our created alerts
		s.Contains(alertIDs, alert.GetId())
	}
	s.Len(returnedIDs, 2) // Ensure no duplicates

	// Test 3: Search with offset and limit (skip first 2, get next 2)
	queryWithOffsetLimit := &v1.Query{
		Pagination: &v1.QueryPagination{
			Offset: 2,
			Limit:  2,
		},
	}
	listAlerts, err = s.datastore.SearchListAlerts(ctx, queryWithOffsetLimit, true)
	s.NoError(err)
	s.Len(listAlerts, 2)

	// Verify these are different alerts from the first page
	offsetReturnedIDs := make(map[string]bool)
	for _, alert := range listAlerts {
		offsetReturnedIDs[alert.GetId()] = true
		s.Contains(alertIDs, alert.GetId())
		// Ensure these are different from the first page
		s.False(returnedIDs[alert.GetId()], "Alert %s should not appear in both pages", alert.GetId())
	}
	s.Len(offsetReturnedIDs, 2)

	// Test 4: Search with large offset (should return remaining alerts)
	queryWithLargeOffset := &v1.Query{
		Pagination: &v1.QueryPagination{
			Offset: 4,
			Limit:  10, // More than remaining
		},
	}
	listAlerts, err = s.datastore.SearchListAlerts(ctx, queryWithLargeOffset, true)
	s.NoError(err)
	s.Len(listAlerts, 1) // Only 1 alert remaining after offset 4

	// Test 5: Search with offset beyond available results
	queryBeyondResults := &v1.Query{
		Pagination: &v1.QueryPagination{
			Offset: 10, // Beyond our 5 alerts
			Limit:  5,
		},
	}
	listAlerts, err = s.datastore.SearchListAlerts(ctx, queryBeyondResults, true)
	s.NoError(err)
	s.Len(listAlerts, 0) // Should return empty

	// Test 6: Verify content of returned alerts matches expected structure
	firstPageQuery := &v1.Query{
		Pagination: &v1.QueryPagination{
			Limit: 1,
		},
	}
	listAlerts, err = s.datastore.SearchListAlerts(ctx, firstPageQuery, true)
	s.NoError(err)
	s.Len(listAlerts, 1)

	// Find the corresponding created alert and verify conversion
	returnedAlert := listAlerts[0]
	var matchingCreatedAlert *storage.Alert
	for _, created := range createdAlerts {
		if created.GetId() == returnedAlert.GetId() {
			matchingCreatedAlert = created
			break
		}
	}
	s.NotNil(matchingCreatedAlert, "Should find matching created alert")

	expectedListAlert := convert.AlertToListAlert(matchingCreatedAlert)
	protoassert.Equal(s.T(), expectedListAlert, returnedAlert)
}

// TestConvertAlert covers the same functionality as searcher_impl_test.go TestConvertAlert
func (s *AlertDatastoreImplSuite) TestConvertAlert() {
	nonNamespacedResourceAlert := fixtures.GetResourceAlert()
	nonNamespacedResourceAlert.GetResource().Namespace = ""

	for _, testCase := range []struct {
		desc             string
		alert            *storage.ListAlert
		expectedLocation string
	}{
		{
			desc:             "Deployment alert",
			alert:            convert.AlertToListAlert(fixtures.GetAlert()),
			expectedLocation: "/prod cluster/stackrox/Deployment/nginx_server",
		},
		{
			desc:             "Namespaced resource alert",
			alert:            convert.AlertToListAlert(fixtures.GetResourceAlert()),
			expectedLocation: "/prod cluster/stackrox/Secrets/my-secret",
		},
		{
			desc:             "Non-namespaced resource alert",
			alert:            convert.AlertToListAlert(nonNamespacedResourceAlert),
			expectedLocation: "/prod cluster/Secrets/my-secret",
		},
	} {
		s.T().Run(testCase.desc, func(t *testing.T) {
			res := convertAlert(testCase.alert, search.Result{})
			assert.Equal(t, testCase.expectedLocation, res.GetLocation())
		})
	}
}

// TestCountAlerts tests the CountAlerts functionality
func (s *AlertDatastoreImplSuite) TestCountAlerts() {
	// Create some active alerts using Alert constants
	activeAlertIDs := []string{fixtureconsts.Alert1, fixtureconsts.Alert2, fixtureconsts.Alert3}
	for _, id := range activeAlertIDs {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.State = storage.ViolationState_ACTIVE
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false

		s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
		s.createAndTrackAlert(alert)
	}

	// Create a resolved alert using Alert constant
	resolvedAlert := fixtures.GetAlert()
	resolvedAlert.Id = fixtureconsts.Alert4
	resolvedAlert.State = storage.ViolationState_RESOLVED
	resolvedAlert.EntityType = storage.Alert_DEPLOYMENT
	resolvedAlert.PlatformComponent = false

	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
	s.createAndTrackAlert(resolvedAlert)

	// Test CountAlerts - should only count active alerts
	count, err := s.datastore.CountAlerts(ctx)
	s.NoError(err)
	s.Equal(len(activeAlertIDs), count)
}

// TestMarkAlertsResolvedBatch tests the batch resolution functionality
func (s *AlertDatastoreImplSuite) TestMarkAlertsResolvedBatch() {
	// Create multiple active alerts
	alertIDs := []string{fixtureconsts.Alert1, fixtureconsts.Alert2, fixtureconsts.Alert3}

	for _, id := range alertIDs {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.State = storage.ViolationState_ACTIVE
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false

		s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
		s.NoError(s.datastore.UpsertAlert(ctx, alert))
	}

	// Mock platform matcher for the resolution process
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil).Times(len(alertIDs))

	// Mark alerts as resolved
	resolvedAlerts, err := s.datastore.MarkAlertsResolvedBatch(ctx, alertIDs...)
	s.NoError(err)
	s.Len(resolvedAlerts, len(alertIDs))

	// Verify all alerts are resolved
	for _, resolvedAlert := range resolvedAlerts {
		s.Equal(storage.ViolationState_RESOLVED, resolvedAlert.GetState())
		s.NotNil(resolvedAlert.GetResolvedAt())
	}

	// Verify alerts are actually resolved in storage
	for _, id := range alertIDs {
		alert, exists, err := s.datastore.GetAlert(ctx, id)
		s.NoError(err)
		s.True(exists)
		s.Equal(storage.ViolationState_RESOLVED, alert.GetState())
	}
}

// TestDeleteAlerts tests the delete functionality
func (s *AlertDatastoreImplSuite) TestDeleteAlerts() {
	// Create alerts
	alertIDs := []string{fixtureconsts.Alert1, fixtureconsts.Alert2}

	for _, id := range alertIDs {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false

		s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
		s.NoError(s.datastore.UpsertAlert(ctx, alert))
	}

	// Verify alerts exist
	for _, id := range alertIDs {
		_, exists, err := s.datastore.GetAlert(ctx, id)
		s.NoError(err)
		s.True(exists)
	}

	// Delete alerts
	err := s.datastore.DeleteAlerts(ctx, alertIDs...)
	s.NoError(err)

	// Verify alerts are deleted
	for _, id := range alertIDs {
		_, exists, err := s.datastore.GetAlert(ctx, id)
		s.NoError(err)
		s.False(exists)
	}
}

// TestWalkByQuery tests the WalkByQuery functionality
func (s *AlertDatastoreImplSuite) TestWalkByQuery() {
	// Create alerts
	alertIDs := []string{fixtureconsts.Alert1, fixtureconsts.Alert2}

	for _, id := range alertIDs {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false

		s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
		s.NoError(s.datastore.UpsertAlert(ctx, alert))
	}

	// Walk by query and collect alerts
	var walkedAlerts []*storage.Alert
	err := s.datastore.WalkByQuery(ctx, search.EmptyQuery(), func(alert *storage.Alert) error {
		walkedAlerts = append(walkedAlerts, alert)
		return nil
	})
	s.NoError(err)
	s.Len(walkedAlerts, len(alertIDs))

	// Verify walked alerts contain our created alerts
	walkedIDs := make(map[string]bool)
	for _, alert := range walkedAlerts {
		walkedIDs[alert.GetId()] = true
	}
	for _, id := range alertIDs {
		s.True(walkedIDs[id])
	}
}

// TestWalkAll tests the WalkAll functionality
func (s *AlertDatastoreImplSuite) TestWalkAll() {
	// Create alerts
	alertIDs := []string{fixtureconsts.Alert1, fixtureconsts.Alert2}

	for _, id := range alertIDs {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false

		s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
		s.NoError(s.datastore.UpsertAlert(ctx, alert))
	}

	// Walk all alerts and collect them
	var walkedAlerts []*storage.ListAlert
	err := s.datastore.WalkAll(ctx, func(listAlert *storage.ListAlert) error {
		walkedAlerts = append(walkedAlerts, listAlert)
		return nil
	})
	s.NoError(err)
	s.Len(walkedAlerts, len(alertIDs))

	// Verify walked alerts contain our created alerts
	walkedIDs := make(map[string]bool)
	for _, alert := range walkedAlerts {
		walkedIDs[alert.GetId()] = true
	}
	for _, id := range alertIDs {
		s.True(walkedIDs[id])
	}
}

// TestUpsert_PlatformComponentAndEntityTypeAssignment tests platform component assignment logic
// Moved from datastore_test.go and converted to use real data instead of mocks
func (s *AlertDatastoreImplSuite) TestUpsert_PlatformComponentAndEntityTypeAssignment() {
	s.T().Setenv(features.PlatformComponents.EnvVar(), "true")
	if !features.PlatformComponents.Enabled() {
		s.T().Skip("Skip test when ROX_PLATFORM_COMPONENTS disabled")
		s.T().SkipNow()
	}

	// Test Case 1: Resource alert
	resourceAlert := &storage.Alert{
		Id: fixtureconsts.AlertFake,
		Entity: &storage.Alert_Resource_{Resource: &storage.Alert_Resource{
			Name:         "test-secret",
			ClusterId:    fixtureconsts.Cluster1,
			Namespace:    "test-namespace",
			ResourceType: storage.Alert_Resource_SECRETS,
		}},
		Policy: &storage.Policy{
			Id:   "policy-1",
			Name: "Test Policy",
		},
		State: storage.ViolationState_ACTIVE,
	}

	// Mock platform matcher to return false (not a platform component)
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
	s.createAndTrackAlert(resourceAlert)

	// Verify alert was stored with correct entity type and platform component flag
	storedAlert, exists, err := s.datastore.GetAlert(ctx, resourceAlert.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(storage.Alert_RESOURCE, storedAlert.GetEntityType())
	s.False(storedAlert.GetPlatformComponent())

	// Test Case 2: Container image alert
	imageAlert := &storage.Alert{
		Id: fixtureconsts.Role1,
		Entity: &storage.Alert_Image{Image: &storage.ContainerImage{
			Id: "image-id",
			Name: &storage.ImageName{
				FullName: "nginx:latest",
			},
		}},
		Policy: &storage.Policy{
			Id:   "policy-2",
			Name: "Test Policy 2",
		},
		State: storage.ViolationState_ACTIVE,
	}

	// Mock platform matcher to return false (not a platform component)
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
	s.createAndTrackAlert(imageAlert)

	// Verify alert was stored with correct entity type and platform component flag
	storedAlert, exists, err = s.datastore.GetAlert(ctx, imageAlert.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(storage.Alert_CONTAINER_IMAGE, storedAlert.GetEntityType())
	s.False(storedAlert.GetPlatformComponent())

	// Test Case 3: Deployment alert not matching platform rules
	deploymentAlert := &storage.Alert{
		Id: fixtureconsts.Role2,
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{
			Id:        "deployment-id",
			Name:      "test-deployment",
			Namespace: "my-namespace",
			ClusterId: fixtureconsts.Cluster1,
		}},
		Policy: &storage.Policy{
			Id:   "policy-3",
			Name: "Test Policy 3",
		},
		State: storage.ViolationState_ACTIVE,
	}

	// Mock platform matcher to return false (not matching platform rules)
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(false, nil)
	s.createAndTrackAlert(deploymentAlert)

	// Verify alert was stored with correct entity type and platform component flag
	storedAlert, exists, err = s.datastore.GetAlert(ctx, deploymentAlert.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(storage.Alert_DEPLOYMENT, storedAlert.GetEntityType())
	s.False(storedAlert.GetPlatformComponent())

	// Test Case 4: Deployment alert matching platform rules
	platformDeploymentAlert := &storage.Alert{
		Id: fixtureconsts.Role3,
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{
			Id:        "platform-deployment-id",
			Name:      "openshift-controller",
			Namespace: "openshift-system",
			ClusterId: fixtureconsts.Cluster1,
		}},
		Policy: &storage.Policy{
			Id:   "policy-4",
			Name: "Test Policy 4",
		},
		State: storage.ViolationState_ACTIVE,
	}

	// Mock platform matcher to return true (matching platform rules)
	s.matcher.EXPECT().MatchAlert(gomock.Any()).Return(true, nil)
	s.createAndTrackAlert(platformDeploymentAlert)

	// Verify alert was stored with correct entity type and platform component flag
	storedAlert, exists, err = s.datastore.GetAlert(ctx, platformDeploymentAlert.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(storage.Alert_DEPLOYMENT, storedAlert.GetEntityType())
	s.True(storedAlert.GetPlatformComponent())
}
