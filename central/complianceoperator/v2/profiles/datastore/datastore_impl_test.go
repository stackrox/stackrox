//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	profileSearch "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/search"
	profileStorage "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	profileUID1 = uuid.NewV4().String()
	profileUID2 = uuid.NewV4().String()
)

func TestComplianceProfileDataStore(t *testing.T) {
	suite.Run(t, new(complianceProfileDataStoreTestSuite))
}

type complianceProfileDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	hasReadCtx   context.Context
	hasWriteCtx  context.Context
	noAccessCtx  context.Context
	testContexts map[string]context.Context

	dataStore DataStore
	storage   profileStorage.Store
	db        *pgtest.TestPostgres
}

func (s *complianceProfileDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceProfileDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Compliance)

	s.mockCtrl = gomock.NewController(s.T())

	s.db = pgtest.ForT(s.T())

	s.storage = profileStorage.New(s.db)
	indexer := profileStorage.NewIndexer(s.db)
	searcher := profileSearch.New(s.storage, indexer)
	s.dataStore = GetTestPostgresDataStore(s.T(), s.db, searcher)
}

func (s *complianceProfileDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceProfileDataStoreTestSuite) TestUpsertProfile() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile(profileUID1, "ocp4", "1.2", testconsts.Cluster1, 0)
	rec2 := getTestProfile(profileUID2, "rhcos-moderate", "7.6", testconsts.Cluster1, 0)
	ids := []string{rec1.GetId(), rec2.GetId()}

	// Verify upsert with Cluster 1 access
	s.Require().NoError(s.dataStore.UpsertProfile(s.testContexts[testutils.Cluster1ReadWriteCtx], rec1))
	// Verify upsert with global access
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	// upsert with read context
	s.Require().Error(s.dataStore.UpsertProfile(s.hasReadCtx, rec2))

	// upsert without permissions on the cluster 1 with only cluster 2 access
	s.Require().Error(s.dataStore.UpsertProfile(s.testContexts[testutils.Cluster2ReadWriteCtx], rec2))

	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(rec1, retrieveRec1)
}

func (s *complianceProfileDataStoreTestSuite) TestDeleteProfileForCluster() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile(profileUID1, "ocp4", "1.2", testconsts.Cluster1, 0)
	rec2 := getTestProfile(profileUID2, "rhcos-moderate", "7.6", testconsts.Cluster2, 0)
	ids := []string{rec1.GetId(), rec2.GetId()}

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(rec1, retrieveRec1)

	s.Require().NoError(s.dataStore.DeleteProfileForCluster(s.hasWriteCtx, profileUID1, testconsts.Cluster1))

	profiles, err := s.dataStore.GetProfilesByClusters(s.hasReadCtx, []string{testconsts.Cluster1})
	s.Require().NoError(err)
	s.Require().Equal(0, len(profiles))

	profiles, err = s.dataStore.GetProfilesByClusters(s.hasReadCtx, []string{testconsts.Cluster2})
	s.Require().NoError(err)
	s.Require().Equal(1, len(profiles))
	s.Require().Equal(profileUID2, profiles[0].Id)

	// Without write access
	s.Require().Error(s.dataStore.DeleteProfileForCluster(s.noAccessCtx, profileUID1, testconsts.Cluster1))

	// Without write access to Cluster 1
	s.Require().Error(s.dataStore.DeleteProfileForCluster(s.testContexts[testutils.Cluster2ReadWriteCtx], profileUID1, testconsts.Cluster1))
}

func (s *complianceProfileDataStoreTestSuite) TestGetProfilesByCluster() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile(profileUID1, "ocp4", "1.2", testconsts.Cluster1, 0)
	rec2 := getTestProfile(profileUID2, "rhcos-moderate", "7.6", testconsts.Cluster2, 0)

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2))

	testCases := []struct {
		desc           string
		clusterID      string
		testContext    context.Context
		expectedRecord *storage.ComplianceOperatorProfileV2
	}{
		{
			desc:           "Cluster 1 - Full access",
			clusterID:      testconsts.Cluster1,
			testContext:    s.testContexts[testutils.UnrestrictedReadCtx],
			expectedRecord: rec1,
		},
		{
			desc:           "Cluster 1 - Only cluster 2 access",
			clusterID:      testconsts.Cluster1,
			testContext:    s.testContexts[testutils.Cluster2ReadWriteCtx],
			expectedRecord: nil,
		},
		{
			desc:           "Cluster 2 query - Only cluster 2 access",
			clusterID:      testconsts.Cluster2,
			testContext:    s.testContexts[testutils.Cluster2ReadWriteCtx],
			expectedRecord: rec2,
		},
		{
			desc:           "Cluster 3 query - Cluster 1 and 2 access",
			clusterID:      testconsts.Cluster3,
			testContext:    s.testContexts[testutils.UnrestrictedReadWriteCtx],
			expectedRecord: nil,
		},
	}

	for _, tc := range testCases {
		profiles, err := s.dataStore.GetProfilesByClusters(tc.testContext, []string{tc.clusterID})
		s.Require().NoError(err)
		if tc.expectedRecord == nil {
			s.Require().Equal(0, len(profiles))
		} else {
			s.Require().Contains(profiles, tc.expectedRecord)
			s.Require().Equal(1, len(profiles))
		}

	}
}

func (s *complianceProfileDataStoreTestSuite) TestGetProfile() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile(profileUID1, "ocp4", "1.2", testconsts.Cluster1, 0)
	rec2 := getTestProfile(profileUID2, "rhcos-moderate", "7.6", testconsts.Cluster1, 0)
	records := map[string]*storage.ComplianceOperatorProfileV2{rec1.GetId(): rec1, rec2.GetId(): rec2}

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2))

	for profileID, profile := range records {
		returnedProfile, found, err := s.dataStore.GetProfile(s.testContexts[testutils.Cluster1ReadWriteCtx], profileID)
		s.Require().NoError(err)
		s.Require().True(found)
		s.Require().Equal(profile, returnedProfile)
	}

	// Test with no access to cluster 1
	_, found, err := s.dataStore.GetProfile(s.testContexts[testutils.Cluster2ReadWriteCtx], rec1.GetProfileId())
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *complianceProfileDataStoreTestSuite) TestSearchProfiles() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile(profileUID1, "ocp4", "1.2", testconsts.Cluster1, 0)
	rec2 := getTestProfile(profileUID2, "rhcos-moderate", "7.6", testconsts.Cluster1, 0)

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2))

	returnedProfiles, err := s.dataStore.SearchProfiles(s.testContexts[testutils.Cluster1ReadWriteCtx], search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorProfileName, rec1.GetName()).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, len(returnedProfiles))
	s.Require().Contains(returnedProfiles, rec1)

	returnedProfiles, err = s.dataStore.SearchProfiles(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorProfileName, "bogus name").ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, len(returnedProfiles))

	// Test with no access
	returnedProfiles, err = s.dataStore.SearchProfiles(s.testContexts[testutils.Cluster2ReadWriteCtx], search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorProfileName, rec1.GetName()).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Empty(returnedProfiles)
}

func (s *complianceProfileDataStoreTestSuite) TestGetProfileCount() {
	rec1 := getTestProfile(profileUID1, "ocp4", "1.2", testconsts.Cluster1, 0)
	rec2 := getTestProfile(profileUID2, "rhcos-moderate", "7.6", testconsts.Cluster1, 0)

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2))

	q := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, rec1.GetName()).ProtoQuery()
	count, err := s.dataStore.CountProfiles(s.hasReadCtx, q)
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	// Empty query to get them all
	count, err = s.dataStore.CountProfiles(s.hasReadCtx, search.NewQueryBuilder().ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, count)
}

func getTestProfile(profileUID string, profileName string, version string, clusterID string, ruleCount int) *storage.ComplianceOperatorProfileV2 {
	var rules []*storage.ComplianceOperatorProfileV2_Rule

	if ruleCount > 0 {
		rules = make([]*storage.ComplianceOperatorProfileV2_Rule, 0, ruleCount)
		for i := 0; i < ruleCount; i++ {
			rules = append(rules, &storage.ComplianceOperatorProfileV2_Rule{
				RuleName: fmt.Sprintf("name-%d", i),
			})
		}
	}

	return &storage.ComplianceOperatorProfileV2{
		Id:             profileUID,
		ProfileId:      uuid.NewV4().String(),
		Name:           profileName,
		ProfileVersion: version,
		ProductType:    "platform",
		Standard:       profileName,
		Description:    "this is a test",
		Labels:         nil,
		Annotations:    nil,
		Product:        "test",
		ClusterId:      clusterID,
		Title:          "A Title",
		Rules:          rules,
	}
}
