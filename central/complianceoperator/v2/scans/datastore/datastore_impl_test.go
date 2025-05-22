//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	profileStorage "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	profileSearch "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/search"
	profilePostgresStorage "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	scanConfigStorage "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
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

func (s *complianceScanDataStoreTestSuite) TestGetProfileScanNames() {
	// Create Test DataStores for ScanConfigurations and Profiles
	scanConfigDS := scanConfigStorage.GetTestPostgresDataStore(s.T(), s.db)
	profilePostgres := profilePostgresStorage.New(s.db)
	searcher := profileSearch.New(profilePostgres)
	profileDS := profileStorage.GetTestPostgresDataStore(s.T(), s.db, searcher)

	profileName1 := "ocp4-cis"
	profileName2 := "ocp4-cis-node"

	// Define Profiles 1 and 2 in Cluster 1
	profile1 := getTestProfile(profileName1, fixtureconsts.ComplianceProfileID1, testconsts.Cluster1)
	profile2 := getTestProfile(profileName2, fixtureconsts.ComplianceProfileID2, testconsts.Cluster1)

	// Define Profiles 1 and 2 in Cluster 2
	profile3 := getTestProfile(profileName1, fixtureconsts.ComplianceProfileID1, testconsts.Cluster2)
	profile4 := getTestProfile(profileName2, fixtureconsts.ComplianceProfileID2, testconsts.Cluster2)

	// Define ScanConfiguration
	scanConfigName := "scanConfig1"
	scanConfig := getTestScanConfig(scanConfigName,
		fixtureconsts.ComplianceScanConfigID1,
		[]string{testconsts.Cluster1, testconsts.Cluster2},
		[]string{profileName1, profileName2})

	s.T().Cleanup(func() {
		_, _ = scanConfigDS.DeleteScanConfiguration(s.hasWriteCtx, fixtureconsts.ComplianceScanConfigID1)
		_ = profileDS.DeleteProfilesByCluster(s.hasWriteCtx, fixtureconsts.Cluster1)
		_ = profileDS.DeleteProfilesByCluster(s.hasWriteCtx, fixtureconsts.Cluster2)
	})

	// Create Profiles 1 and 2 in Cluster 1
	s.Require().NoError(profileDS.UpsertProfile(s.hasWriteCtx, profile1))
	s.Require().NoError(profileDS.UpsertProfile(s.hasWriteCtx, profile2))

	// Create Profiles 1 and 2 in Cluster 2
	s.Require().NoError(profileDS.UpsertProfile(s.hasWriteCtx, profile3))
	s.Require().NoError(profileDS.UpsertProfile(s.hasWriteCtx, profile4))

	// Create ScanConfiguration
	s.Require().NoError(scanConfigDS.UpsertScanConfiguration(s.hasWriteCtx, scanConfig))

	// make sure we have nothing
	ScanIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(ScanIDs)

	scanName1 := "ocp4-cis"
	scanName2 := "ocp4-cis-node-worker"
	scanName3 := "ocp4-cis-node-master"

	// Define Scan 1, 2, and 3 for Cluster 1
	testScan1 := getTestScanWithScanConfig(scanName1, scanConfigName, fixtureconsts.ComplianceProfileID1, testconsts.Cluster1)
	testScan2 := getTestScanWithScanConfig(scanName2, scanConfigName, fixtureconsts.ComplianceProfileID2, testconsts.Cluster1)
	testScan3 := getTestScanWithScanConfig(scanName3, scanConfigName, fixtureconsts.ComplianceProfileID2, testconsts.Cluster1)

	// Define Scan 1, 2, and 3 for Cluster 2
	testScan4 := getTestScanWithScanConfig(scanName1, scanConfigName, fixtureconsts.ComplianceProfileID1, testconsts.Cluster2)
	testScan5 := getTestScanWithScanConfig(scanName2, scanConfigName, fixtureconsts.ComplianceProfileID2, testconsts.Cluster2)
	testScan6 := getTestScanWithScanConfig(scanName3, scanConfigName, fixtureconsts.ComplianceProfileID2, testconsts.Cluster2)

	s.T().Cleanup(func() {
		_ = s.dataStore.DeleteScan(s.hasWriteCtx, testScan1.GetId())
		_ = s.dataStore.DeleteScan(s.hasWriteCtx, testScan2.GetId())
		_ = s.dataStore.DeleteScan(s.hasWriteCtx, testScan3.GetId())
		_ = s.dataStore.DeleteScan(s.hasWriteCtx, testScan4.GetId())
		_ = s.dataStore.DeleteScan(s.hasWriteCtx, testScan5.GetId())
		_ = s.dataStore.DeleteScan(s.hasWriteCtx, testScan6.GetId())
	})

	// Create Scans
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan1))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan2))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan3))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan4))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan5))
	s.Require().NoError(s.dataStore.UpsertScan(s.hasWriteCtx, testScan6))

	s.Run("get scan to profiles from cluster 1", func() {
		scanToProfileMap, err := s.dataStore.GetProfilesScanNamesByScanConfigAndCluster(s.hasReadCtx, fixtureconsts.ComplianceScanConfigID1, testconsts.Cluster1)
		s.Require().NoError(err)
		s.Require().NotNil(scanToProfileMap)

		s.Assert().Contains(scanToProfileMap, scanName1)
		s.Assert().Contains(scanToProfileMap, scanName2)
		s.Assert().Contains(scanToProfileMap, scanName3)

		s.Assert().Equal(profileName1, scanToProfileMap[scanName1])
		s.Assert().Equal(fmt.Sprintf("%s-worker", profileName2), scanToProfileMap[scanName2])
		s.Assert().Equal(fmt.Sprintf("%s-master", profileName2), scanToProfileMap[scanName3])
	})

	s.Run("get scan to profiles from cluster 2", func() {
		scanToProfileMap, err := s.dataStore.GetProfilesScanNamesByScanConfigAndCluster(s.hasReadCtx, fixtureconsts.ComplianceScanConfigID1, testconsts.Cluster2)
		s.Require().NoError(err)
		s.Require().NotNil(scanToProfileMap)

		s.Assert().Contains(scanToProfileMap, scanName1)
		s.Assert().Contains(scanToProfileMap, scanName2)
		s.Assert().Contains(scanToProfileMap, scanName3)

		s.Assert().Equal(profileName1, scanToProfileMap[scanName1])
		s.Assert().Equal(fmt.Sprintf("%s-worker", profileName2), scanToProfileMap[scanName2])
		s.Assert().Equal(fmt.Sprintf("%s-master", profileName2), scanToProfileMap[scanName3])
	})

	s.Run("get empty scan to profiles (wrong cluster)", func() {
		scanToProfileMap, err := s.dataStore.GetProfilesScanNamesByScanConfigAndCluster(s.hasReadCtx, fixtureconsts.ComplianceScanConfigID1, testconsts.WrongCluster)
		s.Require().NoError(err)
		s.Require().NotNil(scanToProfileMap)
		s.Assert().Len(scanToProfileMap, 0)
	})

	s.Run("get empty scan to profiles (wrong scan config)", func() {
		scanToProfileMap, err := s.dataStore.GetProfilesScanNamesByScanConfigAndCluster(s.hasReadCtx, fixtureconsts.ComplianceScanConfigID2, testconsts.Cluster1)
		s.Require().NoError(err)
		s.Require().NotNil(scanToProfileMap)
		s.Assert().Len(scanToProfileMap, 0)
	})

	s.Run("get scan to profiles with profile ref 1", func() {
		scanToProfileMap, err := s.dataStore.GetProfileScanNamesByScanConfigClusterAndProfileRef(s.hasReadCtx, fixtureconsts.ComplianceScanConfigID1, testconsts.Cluster1, []string{fixtureconsts.ComplianceProfileID1})
		s.Require().NoError(err)
		s.Require().NotNil(scanToProfileMap)

		s.Assert().Contains(scanToProfileMap, scanName1)
		s.Assert().NotContains(scanToProfileMap, scanName2)
		s.Assert().NotContains(scanToProfileMap, scanName3)

		s.Assert().Equal(profileName1, scanToProfileMap[scanName1])
		s.Assert().Len(scanToProfileMap[scanName2], 0)
		s.Assert().Len(scanToProfileMap[scanName3], 0)
	})

	s.Run("get scan to profiles with profile ref 2", func() {
		scanToProfileMap, err := s.dataStore.GetProfileScanNamesByScanConfigClusterAndProfileRef(s.hasReadCtx, fixtureconsts.ComplianceScanConfigID1, testconsts.Cluster1, []string{fixtureconsts.ComplianceProfileID2})
		s.Require().NoError(err)
		s.Require().NotNil(scanToProfileMap)

		s.Assert().NotContains(scanToProfileMap, scanName1)
		s.Assert().Contains(scanToProfileMap, scanName2)
		s.Assert().Contains(scanToProfileMap, scanName3)

		s.Assert().Len(scanToProfileMap[scanName1], 0)
		s.Assert().Equal(fmt.Sprintf("%s-worker", profileName2), scanToProfileMap[scanName2])
		s.Assert().Equal(fmt.Sprintf("%s-master", profileName2), scanToProfileMap[scanName3])
	})

	s.Run("get scan to profiles with profile ref 1 and 2", func() {
		scanToProfileMap, err := s.dataStore.GetProfileScanNamesByScanConfigClusterAndProfileRef(s.hasReadCtx, fixtureconsts.ComplianceScanConfigID1, testconsts.Cluster1, []string{fixtureconsts.ComplianceProfileID1, fixtureconsts.ComplianceProfileID2})
		s.Require().NoError(err)
		s.Require().NotNil(scanToProfileMap)

		s.Assert().Contains(scanToProfileMap, scanName1)
		s.Assert().Contains(scanToProfileMap, scanName2)
		s.Assert().Contains(scanToProfileMap, scanName3)

		s.Assert().Equal(profileName1, scanToProfileMap[scanName1])
		s.Assert().Equal(fmt.Sprintf("%s-worker", profileName2), scanToProfileMap[scanName2])
		s.Assert().Equal(fmt.Sprintf("%s-master", profileName2), scanToProfileMap[scanName3])
	})
}

func getTestScan(scanName string, profileID string, clusterID string) *storage.ComplianceOperatorScanV2 {
	return getTestScanWithScanConfig(scanName, scanName, profileID, clusterID)
}

func getTestScanWithScanConfig(scanName string, scanConfig string, profileID string, clusterID string) *storage.ComplianceOperatorScanV2 {
	return &storage.ComplianceOperatorScanV2{
		Id:             uuid.NewV4().String(),
		ScanConfigName: scanConfig,
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

func getTestProfile(profileName string, profileRefID string, clusterID string) *storage.ComplianceOperatorProfileV2 {
	return &storage.ComplianceOperatorProfileV2{
		Id:           uuid.NewV4().String(),
		Name:         profileName,
		ProfileRefId: profileRefID,
		ClusterId:    clusterID,
	}
}

func getTestScanConfig(scanConfigName string, scanConfigID string, clusterID []string, profiles []string) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:             scanConfigID,
		ScanConfigName: scanConfigName,
		Profiles: func() []*storage.ComplianceOperatorScanConfigurationV2_ProfileName {
			var ret []*storage.ComplianceOperatorScanConfigurationV2_ProfileName
			for _, profile := range profiles {
				ret = append(ret, &storage.ComplianceOperatorScanConfigurationV2_ProfileName{
					ProfileName: profile,
				})
			}
			return ret
		}(),
		Clusters: func() []*storage.ComplianceOperatorScanConfigurationV2_Cluster {
			var ret []*storage.ComplianceOperatorScanConfigurationV2_Cluster
			for _, cluster := range clusterID {
				ret = append(ret, &storage.ComplianceOperatorScanConfigurationV2_Cluster{
					ClusterId: cluster,
				})
			}
			return ret
		}(),
	}
}
