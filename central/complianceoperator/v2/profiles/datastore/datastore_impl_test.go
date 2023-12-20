//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	profileSearch "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/search"
	profileEdgeStorage "github.com/stackrox/rox/central/complianceoperator/v2/profiles/profileclusteredge/store/postgres"
	profileStorage "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestComplianceProfileDataStore(t *testing.T) {
	suite.Run(t, new(complianceProfileDataStoreTestSuite))
}

type complianceProfileDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	noAccessCtx context.Context

	dataStore   DataStore
	storage     profileStorage.Store
	edgeStorage profileEdgeStorage.Store
	db          *pgtest.TestPostgres
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
			sac.ResourceScopeKeys(resources.ComplianceOperator)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ComplianceOperator)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.mockCtrl = gomock.NewController(s.T())

	s.db = pgtest.ForT(s.T())

	s.storage = profileStorage.New(s.db)
	s.edgeStorage = profileEdgeStorage.New(s.db)
	indexer := profileStorage.NewIndexer(s.db)
	search := profileSearch.New(s.storage, indexer)
	s.dataStore = New(s.storage, s.edgeStorage, s.db, search)
}

func (s *complianceProfileDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceProfileDataStoreTestSuite) TestUpsertProfile() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile("ocp4", "1.2")
	rec2 := getTestProfile("rhcos-moderate", "7.6")
	ids := []string{rec1.GetId(), rec2.GetId()}

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1, fixtureconsts.Cluster1, uuid.NewV4().String()))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2, fixtureconsts.Cluster1, uuid.NewV4().String()))

	// Add an existing profile to another cluster
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2, fixtureconsts.Cluster2, uuid.NewV4().String()))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	// upsert with read context
	s.Require().Error(s.dataStore.UpsertProfile(s.hasReadCtx, rec2, fixtureconsts.Cluster1, uuid.NewV4().String()))

	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(rec1, retrieveRec1)

	edgeRecs, err := s.edgeStorage.GetByQuery(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, len(edgeRecs))
}

func (s *complianceProfileDataStoreTestSuite) TestDeleteProfileForCluster() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile("ocp4", "1.2")
	rec2 := getTestProfile("rhcos-moderate", "7.6")
	ids := []string{rec1.GetId(), rec2.GetId()}

	profileUID1 := uuid.NewV4().String()
	profileUID2 := uuid.NewV4().String()
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1, fixtureconsts.Cluster1, profileUID1))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2, fixtureconsts.Cluster2, profileUID2))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(rec1, retrieveRec1)

	edgeRecs, err := s.edgeStorage.GetByQuery(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, len(edgeRecs))

	s.Require().NoError(s.dataStore.DeleteProfileForCluster(s.hasWriteCtx, profileUID1, fixtureconsts.Cluster1))

	edgeRecs, err = s.edgeStorage.GetByQuery(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, len(edgeRecs))

	edgeRecs, err = s.edgeStorage.GetByQuery(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, fixtureconsts.Cluster2).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, len(edgeRecs))
	s.Require().Equal(profileUID2, edgeRecs[0].ProfileUid)

	// Without write access
	s.Require().Error(s.dataStore.DeleteProfileForCluster(s.hasReadCtx, profileUID1, fixtureconsts.Cluster1))
}

func (s *complianceProfileDataStoreTestSuite) TestGetProfileEdgesByCluster() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	edgeIDs, err := s.edgeStorage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(edgeIDs)

	rec1 := getTestProfile("ocp4", "1.2")
	rec2 := getTestProfile("rhcos-moderate", "7.6")

	profileUID1 := uuid.NewV4().String()
	profileUID2 := uuid.NewV4().String()
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1, fixtureconsts.Cluster1, profileUID1))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2, fixtureconsts.Cluster2, profileUID2))

	edgeRecs, err := s.dataStore.GetProfileEdgesByCluster(s.hasReadCtx, fixtureconsts.Cluster2)
	s.Require().NoError(err)
	s.Require().Equal(1, len(edgeRecs))
	s.Require().Equal(profileUID2, edgeRecs[0].ProfileUid)

	// Test with no access
	edgeRecs, err = s.dataStore.GetProfileEdgesByCluster(s.noAccessCtx, fixtureconsts.Cluster2)
	s.Require().Error(err)
	s.Require().Equal(0, len(edgeRecs))
}

func (s *complianceProfileDataStoreTestSuite) TestGetProfile() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile("ocp4", "1.2")
	rec2 := getTestProfile("rhcos-moderate", "7.6")
	records := map[string]*storage.ComplianceOperatorProfileV2{rec1.GetId(): rec1, rec2.GetId(): rec2}

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1, fixtureconsts.Cluster1, uuid.NewV4().String()))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2, fixtureconsts.Cluster1, uuid.NewV4().String()))

	for profileID, profile := range records {
		returnedProfile, found, err := s.dataStore.GetProfile(s.hasReadCtx, profileID)
		s.Require().NoError(err)
		s.Require().True(found)
		s.Require().Equal(profile, returnedProfile)
	}

	// Test with no access
	_, found, err := s.dataStore.GetProfile(s.noAccessCtx, rec1.GetProfileId())
	s.Require().Error(err)
	s.Require().False(found)
}

func (s *complianceProfileDataStoreTestSuite) TestSearchProfiles() {
	// make sure we have nothing
	profileIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(profileIDs)

	rec1 := getTestProfile("ocp4", "1.2")
	rec2 := getTestProfile("rhcos-moderate", "7.6")

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1, fixtureconsts.Cluster1, uuid.NewV4().String()))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2, fixtureconsts.Cluster1, uuid.NewV4().String()))

	returnedProfiles, err := s.dataStore.SearchProfiles(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorProfileName, rec1.GetName()).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, len(returnedProfiles))
	s.Require().Contains(returnedProfiles, rec1)

	returnedProfiles, err = s.dataStore.SearchProfiles(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorProfileName, "bogus name").ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, len(returnedProfiles))

	// Test with no access
	returnedProfiles, err = s.dataStore.SearchProfiles(s.noAccessCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorProfileName, rec1.GetName()).ProtoQuery())
	s.Require().Error(err)
	s.Require().Equal(0, len(returnedProfiles))
}

func (s *complianceProfileDataStoreTestSuite) TestGetProfileCount() {
	rec1 := getTestProfile("ocp4", "1.2")
	rec2 := getTestProfile("rhcos-moderate", "7.6")

	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec1, fixtureconsts.Cluster1, uuid.NewV4().String()))
	s.Require().NoError(s.dataStore.UpsertProfile(s.hasWriteCtx, rec2, fixtureconsts.Cluster1, uuid.NewV4().String()))

	q := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, rec1.GetName()).ProtoQuery()
	count, err := s.dataStore.CountProfiles(s.hasReadCtx, q)
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	// Empty query to get them all
	count, err = s.dataStore.CountProfiles(s.hasReadCtx, search.NewQueryBuilder().ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, count)
}

func getTestProfile(profileName string, version string) *storage.ComplianceOperatorProfileV2 {
	return &storage.ComplianceOperatorProfileV2{
		Id:             fmt.Sprintf("%s-%s", profileName, version),
		ProfileId:      uuid.NewV4().String(),
		Name:           profileName,
		ProfileVersion: version,
		ProductType:    "platform",
		Standard:       profileName,
		Description:    "this is a test",
		Labels:         nil,
		Annotations:    nil,
		Product:        "test",
	}
}
