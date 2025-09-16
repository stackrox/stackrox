//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	scanStorage "github.com/stackrox/rox/central/complianceoperator/v2/scans/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestComplianceProfileDataStore(t *testing.T) {
	suite.Run(t, new(complianceScanDataStoreTestSuite))
}

type complianceScanDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	hasReadCtx            context.Context
	hasWriteCtx           context.Context
	noAccessCtx           context.Context
	testContexts          map[string]context.Context
	nonComplianceContexts map[string]context.Context

	dataStore DataStore
	storage   scanStorage.Store
	db        *pgtest.TestPostgres
}

func (s *complianceScanDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceScanDataStoreTestSuite) SetupTest() {
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
	s.nonComplianceContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Deployment)

	s.mockCtrl = gomock.NewController(s.T())

	s.db = pgtest.ForT(s.T())

	s.storage = scanStorage.New(s.db)
	s.dataStore = GetTestPostgresDataStore(s.T(), s.db)
}

func (s *complianceScanDataStoreTestSuite) TestGetScan() {
	// make sure we have nothing
	ScanIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(ScanIDs)

	testScan1 := getTestScan("scan1", "profile-1", testconsts.Cluster1)
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan1))

	testCases := []struct {
		desc           string
		scanID         string
		testContext    context.Context
		expectedResult *storage.ComplianceOperatorScanV2
	}{
		{
			desc:           "Scan exists - Full access",
			scanID:         testScan1.GetId(),
			testContext:    s.testContexts[testutils.UnrestrictedReadCtx],
			expectedResult: testScan1,
		},
		{
			desc:           "Scan exists - Cluster 1 access",
			scanID:         testScan1.GetId(),
			testContext:    s.testContexts[testutils.Cluster1ReadWriteCtx],
			expectedResult: testScan1,
		},
		{
			desc:           "Scan exists - Cluster 2 access",
			scanID:         testScan1.GetId(),
			testContext:    s.testContexts[testutils.Cluster2ReadWriteCtx],
			expectedResult: nil,
		},
		{
			desc:           "Scan exist - No compliance access",
			scanID:         testScan1.GetId(),
			testContext:    s.nonComplianceContexts[testutils.UnrestrictedReadCtx],
			expectedResult: nil,
		},
		{
			desc:           "Scan does not exist - Full access",
			scanID:         uuid.NewV4().String(),
			testContext:    s.testContexts[testutils.UnrestrictedReadCtx],
			expectedResult: nil,
		},
	}
	for _, tc := range testCases {
		retrievedObject, found, err := s.dataStore.GetScan(tc.testContext, tc.scanID)
		s.Require().NoError(err)
		s.Require().True(found != (tc.expectedResult == nil))
		protoassert.Equal(s.T(), tc.expectedResult, retrievedObject)
	}
}

func (s *complianceScanDataStoreTestSuite) TestGetScansByCluster() {
	// make sure we have nothing
	ScanIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(ScanIDs)

	testScan1 := getTestScan("scan1", "profile-1", testconsts.Cluster1)
	testScan2 := getTestScan("scan2", "profile-1", testconsts.Cluster1)
	testScan3 := getTestScan("scan3", "profile-1", testconsts.Cluster2)

	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan1))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan2))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan3))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(3, count)

	testCases := []struct {
		desc            string
		clusterID       string
		testContext     context.Context
		expectedResults []*storage.ComplianceOperatorScanV2
		expectedCount   int
	}{
		{
			desc:            "Scans exist - Full access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[testutils.UnrestrictedReadCtx],
			expectedResults: []*storage.ComplianceOperatorScanV2{testScan1, testScan2},
			expectedCount:   2,
		},
		{
			desc:            "Scans exist - Cluster 1 access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[testutils.Cluster1ReadWriteCtx],
			expectedResults: []*storage.ComplianceOperatorScanV2{testScan1, testScan2},
			expectedCount:   2,
		},
		{
			desc:            "Scans exist - Cluster 2 access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[testutils.Cluster2ReadWriteCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
		{
			desc:            "Scans exists - No compliance access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.nonComplianceContexts[testutils.UnrestrictedReadCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
		{
			desc:            "Scan does not exist - Full access",
			clusterID:       fixtureconsts.ClusterFake1,
			testContext:     s.testContexts[testutils.UnrestrictedReadCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
	}
	for _, tc := range testCases {
		retrievedObjects, err := s.dataStore.GetScansByCluster(tc.testContext, tc.clusterID)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, len(retrievedObjects))
		protoassert.SlicesEqual(s.T(), tc.expectedResults, retrievedObjects)
	}
}

func (s *complianceScanDataStoreTestSuite) TestUpsertScan() {
	// make sure we have nothing
	ScanIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(ScanIDs)

	testScan1 := getTestScan("scan1", "profile-1", testconsts.Cluster1)
	testScan2 := getTestScan("scan2", "profile-1", testconsts.Cluster1)
	testScan3 := getTestScan("scan3", "profile-1", testconsts.Cluster2)

	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan1))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan2))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan3))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(3, count)

	s.Require().Error(s.dataStore.UpsertScan(s.hasReadCtx, testScan3))

	// Update an object
	testScan3.LastExecutedTime = protocompat.TimestampNow()
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan3))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(3, count)

	retrievedObject, found, err := s.dataStore.GetScan(s.hasReadCtx, testScan3.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal(testScan3.LastExecutedTime.AsTime(), retrievedObject.LastExecutedTime.AsTime())
}

func (s *complianceScanDataStoreTestSuite) TestDeleteScanByCluster() {
	testScan1 := getTestScan("scan1", "profile-1", testconsts.Cluster1)
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan1))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	s.Require().NoError(s.dataStore.DeleteScanByCluster(s.hasWriteCtx, testconsts.Cluster1))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, count)
}

func (s *complianceScanDataStoreTestSuite) TestDeleteScan() {
	// make sure we have nothing
	ScanIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(ScanIDs)

	testScan1 := getTestScan("scan1", "profile-1", testconsts.Cluster1)
	testScan2 := getTestScan("scan2", "profile-1", testconsts.Cluster1)
	testScan3 := getTestScan("scan3", "profile-1", testconsts.Cluster2)

	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan1))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan2))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan3))

	s.Require().NoError(s.dataStore.DeleteScan(s.hasWriteCtx, testScan1.GetId()))
	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	retrievedObject, found, err := s.dataStore.GetScan(s.hasReadCtx, testScan1.GetId())
	s.Require().NoError(err)
	s.Require().False(found)
	s.Require().Empty(retrievedObject)

	// Test no access, object should remain
	s.Require().NoError(s.dataStore.DeleteScan(s.noAccessCtx, testScan2.GetId()))
	retrievedObject, found, err = s.dataStore.GetScan(s.hasReadCtx, testScan2.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	protoassert.Equal(s.T(), testScan2, retrievedObject)
}

func getTestScan(scanName string, profileID string, clusterID string) *storage.ComplianceOperatorScanV2 {
	return &storage.ComplianceOperatorScanV2{
		Id:             uuid.NewV4().String(),
		ScanConfigName: scanName,
		ScanName:       scanName,
		ClusterId:      clusterID,
		Errors:         "",
		Warnings:       "",
		Profile: &storage.ProfileShim{
			ProfileRefId: profileID,
		},
		Labels:           nil,
		Annotations:      nil,
		ScanType:         0,
		NodeSelector:     0,
		Status:           nil,
		CreatedTime:      protocompat.TimestampNow(),
		LastExecutedTime: protocompat.TimestampNow(),
	}
}
