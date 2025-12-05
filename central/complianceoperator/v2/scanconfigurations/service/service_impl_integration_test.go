//go:build sql_integration

package service

import (
	"context"
	"testing"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	clusterStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	benchmarkDatastore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	benchmarkStore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	profileStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	scanConfigDatastore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	scanStatusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	scanConfigStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestComplianceScanConfigServiceIntegration(t *testing.T) {
	suite.Run(t, new(ComplianceScanConfigServiceIntegrationTestSuite))
}

type ComplianceScanConfigServiceIntegrationTestSuite struct {
	suite.Suite
	db           *pgtest.TestPostgres
	scanConfigDS scanConfigDatastore.DataStore
	profileDS    profileDatastore.DataStore
	benchmarkDS  benchmarkDatastore.DataStore
	clusterDS    clusterDatastore.DataStore
	service      Service
}

func (s *ComplianceScanConfigServiceIntegrationTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *ComplianceScanConfigServiceIntegrationTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())

	// Create a test cluster in the database first, before creating datastores.
	clusterStorage := clusterStore.New(s.db)
	testCluster := &storage.Cluster{
		Id:   fixtureconsts.Cluster1,
		Name: "test-cluster",
	}
	err := clusterStorage.Upsert(sac.WithAllAccess(context.Background()), testCluster)
	s.Require().NoError(err)

	configStorage := scanConfigStore.New(s.db)
	statusStorage := scanStatusStore.New(s.db)
	s.scanConfigDS = scanConfigDatastore.New(configStorage, statusStorage, s.db.DB)

	profileStorage := profileStore.New(s.db)
	s.profileDS = profileDatastore.New(profileStorage, s.db.DB)

	benchmarkStorage := benchmarkStore.New(s.db)
	s.benchmarkDS = benchmarkDatastore.New(benchmarkStorage)

	s.clusterDS, err = clusterDatastore.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)

	s.service = New(s.scanConfigDS, nil, nil, nil, nil, nil, s.profileDS, s.benchmarkDS, s.clusterDS, nil, nil)
}

func (s *ComplianceScanConfigServiceIntegrationTestSuite) TestListComplianceScanConfigProfiles_PaginationTotalCount() {
	ctx := sac.WithAllAccess(context.Background())

	// Create 4 different profiles in the database.
	profiles := []*storage.ComplianceOperatorProfileV2{
		{
			Id:          uuid.NewV4().String(),
			Name:        "ocp4-cis",
			ClusterId:   fixtureconsts.Cluster1,
			ProfileId:   "ocp4-cis-id",
			Title:       "OCP4 CIS",
			Description: "CIS Benchmark for OCP4",
			ProductType: "platform",
			Rules: []*storage.ComplianceOperatorProfileV2_Rule{
				{RuleName: "rule1"},
				{RuleName: "rule2"},
			},
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "rhcos-moderate",
			ClusterId:   fixtureconsts.Cluster1,
			ProfileId:   "rhcos-moderate-id",
			Title:       "RHCOS Moderate",
			Description: "Moderate security profile for RHCOS",
			ProductType: "node",
			Rules: []*storage.ComplianceOperatorProfileV2_Rule{
				{RuleName: "rule3"},
			},
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "pci-dss",
			ClusterId:   fixtureconsts.Cluster1,
			ProfileId:   "pci-dss-id",
			Title:       "PCI-DSS",
			Description: "PCI-DSS compliance profile",
			ProductType: "platform",
			Rules: []*storage.ComplianceOperatorProfileV2_Rule{
				{RuleName: "rule4"},
				{RuleName: "rule5"},
				{RuleName: "rule6"},
			},
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "nist-800-53",
			ClusterId:   fixtureconsts.Cluster1,
			ProfileId:   "nist-800-53-id",
			Title:       "NIST 800-53",
			Description: "NIST 800-53 security controls",
			ProductType: "platform",
			Rules: []*storage.ComplianceOperatorProfileV2_Rule{
				{RuleName: "rule7"},
			},
		},
	}
	for _, profile := range profiles {
		err := s.profileDS.UpsertProfile(ctx, profile)
		s.Require().NoError(err)
	}

	// Create benchmarks for the profiles.
	benchmarks := []*storage.ComplianceOperatorBenchmarkV2{
		{
			Id:        uuid.NewV4().String(),
			Name:      "CIS Kubernetes",
			ShortName: "CIS_K8S",
			Version:   "1.6.0",
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "NIST",
			ShortName: "NIST",
			Version:   "800-53",
		},
	}
	for _, benchmark := range benchmarks {
		err := s.benchmarkDS.UpsertBenchmark(ctx, benchmark)
		s.Require().NoError(err)
	}

	// Create scan configurations that reference these profiles.
	scanConfigs := []*storage.ComplianceOperatorScanConfigurationV2{
		{
			Id:             uuid.NewV4().String(),
			ScanConfigName: "scan-1",
			Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
				{ProfileName: "ocp4-cis"},
				{ProfileName: "rhcos-moderate"},
			},
			Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
				{ClusterId: fixtureconsts.Cluster1},
			},
		},
		{
			Id:             uuid.NewV4().String(),
			ScanConfigName: "scan-2",
			Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
				{ProfileName: "pci-dss"},
			},
			Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
				{ClusterId: fixtureconsts.Cluster1},
			},
		},
		{
			Id:             uuid.NewV4().String(),
			ScanConfigName: "scan-3",
			Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
				{ProfileName: "nist-800-53"},
			},
			Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
				{ClusterId: fixtureconsts.Cluster1},
			},
		},
	}

	for _, scanConfig := range scanConfigs {
		err := s.scanConfigDS.UpsertScanConfiguration(ctx, scanConfig)
		s.Require().NoError(err)
	}

	testCases := []struct {
		desc                 string
		pagination           *v2.Pagination
		expectedTotalCount   int32
		expectedProfileCount int
	}{
		{
			desc: "Pagination with limit 2, offset 1",
			pagination: &v2.Pagination{
				Limit:  2,
				Offset: 1,
			},
			expectedTotalCount:   4,
			expectedProfileCount: 2,
		},
		{
			desc:                 "No pagination",
			pagination:           nil,
			expectedTotalCount:   4,
			expectedProfileCount: 4,
		},
		{
			desc: "Pagination with limit 1, offset 3",
			pagination: &v2.Pagination{
				Limit:  1,
				Offset: 3,
			},
			expectedTotalCount:   4,
			expectedProfileCount: 1,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			resp, err := s.service.ListComplianceScanConfigProfiles(ctx, &v2.RawQuery{
				Query:      "",
				Pagination: tc.pagination,
			})
			s.Require().NoError(err)
			s.Require().NotNil(resp)

			s.Assert().Equal(tc.expectedTotalCount, resp.GetTotalCount(), "totalCount should match expected")
			s.Assert().Len(resp.GetProfiles(), tc.expectedProfileCount, "profile count should match expected")
		})
	}
}

func (s *ComplianceScanConfigServiceIntegrationTestSuite) TestListComplianceScanConfigClusterProfiles_PaginationTotalCount() {
	ctx := sac.WithAllAccess(context.Background())

	// Cluster was already created in SetupTest.
	// Create 4 different profiles in the database for this cluster.
	profiles := []*storage.ComplianceOperatorProfileV2{
		{
			Id:          uuid.NewV4().String(),
			Name:        "ocp4-cis",
			ClusterId:   fixtureconsts.Cluster1,
			ProfileId:   "ocp4-cis-id",
			Title:       "OCP4 CIS",
			Description: "CIS Benchmark for OCP4",
			ProductType: "platform",
			Rules: []*storage.ComplianceOperatorProfileV2_Rule{
				{RuleName: "rule1"},
				{RuleName: "rule2"},
			},
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "rhcos-moderate",
			ClusterId:   fixtureconsts.Cluster1,
			ProfileId:   "rhcos-moderate-id",
			Title:       "RHCOS Moderate",
			Description: "Moderate security profile for RHCOS",
			ProductType: "node",
			Rules: []*storage.ComplianceOperatorProfileV2_Rule{
				{RuleName: "rule3"},
			},
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "pci-dss",
			ClusterId:   fixtureconsts.Cluster1,
			ProfileId:   "pci-dss-id",
			Title:       "PCI-DSS",
			Description: "PCI-DSS compliance profile",
			ProductType: "platform",
			Rules: []*storage.ComplianceOperatorProfileV2_Rule{
				{RuleName: "rule4"},
				{RuleName: "rule5"},
				{RuleName: "rule6"},
			},
		},
		{
			Id:          uuid.NewV4().String(),
			Name:        "nist-800-53",
			ClusterId:   fixtureconsts.Cluster1,
			ProfileId:   "nist-800-53-id",
			Title:       "NIST 800-53",
			Description: "NIST 800-53 security controls",
			ProductType: "platform",
			Rules: []*storage.ComplianceOperatorProfileV2_Rule{
				{RuleName: "rule7"},
			},
		},
	}
	for _, profile := range profiles {
		err := s.profileDS.UpsertProfile(ctx, profile)
		s.Require().NoError(err)
	}

	// Create benchmarks for the profiles.
	benchmarks := []*storage.ComplianceOperatorBenchmarkV2{
		{
			Id:        uuid.NewV4().String(),
			Name:      "CIS Kubernetes",
			ShortName: "CIS_K8S",
			Version:   "1.6.0",
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "NIST",
			ShortName: "NIST",
			Version:   "800-53",
		},
	}
	for _, benchmark := range benchmarks {
		err := s.benchmarkDS.UpsertBenchmark(ctx, benchmark)
		s.Require().NoError(err)
	}

	// Create scan configurations that reference these profiles.
	scanConfigs := []*storage.ComplianceOperatorScanConfigurationV2{
		{
			Id:             uuid.NewV4().String(),
			ScanConfigName: "cluster-scan-1",
			Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
				{ProfileName: "ocp4-cis"},
				{ProfileName: "rhcos-moderate"},
			},
			Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
				{ClusterId: fixtureconsts.Cluster1},
			},
		},
		{
			Id:             uuid.NewV4().String(),
			ScanConfigName: "cluster-scan-2",
			Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
				{ProfileName: "pci-dss"},
			},
			Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
				{ClusterId: fixtureconsts.Cluster1},
			},
		},
		{
			Id:             uuid.NewV4().String(),
			ScanConfigName: "cluster-scan-3",
			Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
				{ProfileName: "nist-800-53"},
			},
			Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
				{ClusterId: fixtureconsts.Cluster1},
			},
		},
	}

	for _, scanConfig := range scanConfigs {
		err := s.scanConfigDS.UpsertScanConfiguration(ctx, scanConfig)
		s.Require().NoError(err)
	}

	testCases := []struct {
		desc                 string
		pagination           *v2.Pagination
		expectedTotalCount   int32
		expectedProfileCount int
	}{
		{
			desc: "Pagination with limit 2, offset 1",
			pagination: &v2.Pagination{
				Limit:  2,
				Offset: 1,
			},
			expectedTotalCount:   4,
			expectedProfileCount: 2,
		},
		{
			desc:                 "No pagination",
			pagination:           nil,
			expectedTotalCount:   4,
			expectedProfileCount: 4,
		},
		{
			desc: "Pagination with limit 1, offset 3",
			pagination: &v2.Pagination{
				Limit:  1,
				Offset: 3,
			},
			expectedTotalCount:   4,
			expectedProfileCount: 1,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			resp, err := s.service.ListComplianceScanConfigClusterProfiles(ctx, &v2.ComplianceConfigClusterProfileRequest{
				ClusterId: fixtureconsts.Cluster1,
				Query: &v2.RawQuery{
					Query:      "",
					Pagination: tc.pagination,
				},
			})
			s.Require().NoError(err)
			s.Require().NotNil(resp)

			s.Assert().Equal(tc.expectedTotalCount, resp.GetTotalCount(), "totalCount should match expected")
			s.Assert().Len(resp.GetProfiles(), tc.expectedProfileCount, "profile count should match expected")
			s.Assert().Equal(fixtureconsts.Cluster1, resp.GetClusterId(), "cluster ID should match")
			s.Assert().Equal("test-cluster", resp.GetClusterName(), "cluster name should match")
		})
	}
}
