//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	componentStore "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/store/postgres"
	cveStore "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore/store/postgres"
	scanStore "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore/store/postgres"
	vmV2Store "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store"
	vmV2Postgres "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestVMCVEV2DataStore(t *testing.T) {
	suite.Run(t, new(vmCVEV2DataStoreTestSuite))
}

type vmCVEV2DataStoreTestSuite struct {
	suite.Suite

	db          *pgtest.TestPostgres
	datastore   DataStore
	vmStore     vmV2Store.Store
	scanPG      scanStore.Store
	componentPG componentStore.Store
	cvePG       cveStore.Store

	sacCtx   context.Context
	noSacCtx context.Context
}

func (s *vmCVEV2DataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	store := cveStore.New(s.db)
	s.datastore = New(store)
	s.vmStore = vmV2Postgres.New(s.db, concurrency.NewKeyFence())
	s.scanPG = scanStore.New(s.db)
	s.componentPG = componentStore.New(s.db)
	s.cvePG = cveStore.New(s.db)

	s.sacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))
	s.noSacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.DenyAllAccessScopeChecker())
}

func (s *vmCVEV2DataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *vmCVEV2DataStoreTestSuite) createTestVM(clusterID, namespace string, index int) *storage.VirtualMachineV2 {
	vmID := uuid.NewV5FromNonUUIDs(clusterID, fmt.Sprintf("%s-%d", namespace, index)).String()
	return &storage.VirtualMachineV2{
		Id:          vmID,
		Name:        fmt.Sprintf("vm-%d", index),
		Namespace:   namespace,
		ClusterId:   clusterID,
		ClusterName: fmt.Sprintf("cluster-%s", clusterID),
	}
}

func (s *vmCVEV2DataStoreTestSuite) createTestScan(vmID string) *storage.VirtualMachineScanV2 {
	scanID := uuid.NewV5FromNonUUIDs(vmID, "scan").String()
	return &storage.VirtualMachineScanV2{
		Id:       scanID,
		VmV2Id:   vmID,
		ScanOs:   "rhel:9",
		ScanTime: timestamppb.Now(),
		TopCvss:  9.8,
	}
}

func (s *vmCVEV2DataStoreTestSuite) createTestComponent(scanID string) *storage.VirtualMachineComponentV2 {
	componentID := uuid.NewV5FromNonUUIDs(scanID, "component").String()
	return &storage.VirtualMachineComponentV2{
		Id:              componentID,
		VmScanId:        scanID,
		Name:            "openssl",
		Version:         "1.1.1",
		Source:          storage.SourceType_OS,
		OperatingSystem: "rhel:9",
	}
}

func (s *vmCVEV2DataStoreTestSuite) createTestCVE(vmID, componentID string, index int) *storage.VirtualMachineCVEV2 {
	cveID := uuid.NewV5FromNonUUIDs(componentID, fmt.Sprintf("cve-%d", index)).String()
	return &storage.VirtualMachineCVEV2{
		Id:            cveID,
		VmV2Id:        vmID,
		VmComponentId: componentID,
		CveBaseInfo: &storage.CVEInfo{
			Cve:         fmt.Sprintf("CVE-2024-%04d", index),
			PublishedOn: timestamppb.Now(),
			CreatedAt:   timestamppb.Now(),
		},
		PreferredCvss: 7.5,
		Severity:      storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		IsFixable:  true,
		HasFixedBy: &storage.VirtualMachineCVEV2_FixedBy{FixedBy: "1.1.2"},
	}
}

// insertFullChainAndCVE creates a VM -> scan -> component -> CVE chain and returns the CVE.
func (s *vmCVEV2DataStoreTestSuite) insertFullChainAndCVE(clusterID, namespace string, vmIndex, cveIndex int) *storage.VirtualMachineCVEV2 {
	ctx := sac.WithAllAccess(context.Background())
	vm := s.createTestVM(clusterID, namespace, vmIndex)
	s.Require().NoError(s.vmStore.UpsertVM(ctx, vm))
	scan := s.createTestScan(vm.GetId())
	s.Require().NoError(s.scanPG.Upsert(ctx, scan))
	component := s.createTestComponent(scan.GetId())
	s.Require().NoError(s.componentPG.Upsert(ctx, component))
	cve := s.createTestCVE(vm.GetId(), component.GetId(), cveIndex)
	s.Require().NoError(s.cvePG.Upsert(ctx, cve))
	return cve
}

// TestGet verifies basic get functionality.
func (s *vmCVEV2DataStoreTestSuite) TestGet() {
	cve := s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	retrieved, found, err := s.datastore.Get(s.sacCtx, cve.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(cve.GetId(), retrieved.GetId())
	s.Equal(cve.GetCveBaseInfo().GetCve(), retrieved.GetCveBaseInfo().GetCve())
	s.Equal(cve.GetPreferredCvss(), retrieved.GetPreferredCvss())

	// Non-existent CVE
	_, found, err = s.datastore.Get(s.sacCtx, uuid.NewDummy().String())
	s.NoError(err)
	s.False(found)
}

// TestExists verifies existence checks.
func (s *vmCVEV2DataStoreTestSuite) TestExists() {
	cve := s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	exists, err := s.datastore.Exists(s.sacCtx, cve.GetId())
	s.NoError(err)
	s.True(exists)

	exists, err = s.datastore.Exists(s.sacCtx, uuid.NewDummy().String())
	s.NoError(err)
	s.False(exists)
}

// TestCount verifies count functionality.
func (s *vmCVEV2DataStoreTestSuite) TestCount() {
	count, err := s.datastore.Count(s.sacCtx, nil)
	s.NoError(err)
	s.Equal(0, count)

	s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 2, 1)

	count, err = s.datastore.Count(s.sacCtx, nil)
	s.NoError(err)
	s.Equal(2, count)
}

// TestGetBatch verifies batch retrieval.
func (s *vmCVEV2DataStoreTestSuite) TestGetBatch() {
	cve1 := s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	cve2 := s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 2, 1)

	cves, err := s.datastore.GetBatch(s.sacCtx, []string{cve1.GetId(), cve2.GetId()})
	s.NoError(err)
	s.Len(cves, 2)

	fetchedIDs := make([]string, len(cves))
	for i, c := range cves {
		fetchedIDs[i] = c.GetId()
	}
	s.ElementsMatch([]string{cve1.GetId(), cve2.GetId()}, fetchedIDs)
}

// TestSearch verifies search functionality.
func (s *vmCVEV2DataStoreTestSuite) TestSearch() {
	cve1 := s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	cve2 := s.insertFullChainAndCVE(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	results, err := s.datastore.Search(s.sacCtx, nil)
	s.NoError(err)
	s.Len(results, 2)

	fetchedIDs := make([]string, len(results))
	for i, r := range results {
		fetchedIDs[i] = r.ID
	}
	s.ElementsMatch([]string{cve1.GetId(), cve2.GetId()}, fetchedIDs)
}

// TestSearchRawVMCVEs verifies raw search.
func (s *vmCVEV2DataStoreTestSuite) TestSearchRawVMCVEs() {
	s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	s.insertFullChainAndCVE(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	cves, err := s.datastore.SearchRawVMCVEs(s.sacCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(cves, 2)

	for _, c := range cves {
		s.NotEmpty(c.GetId())
		s.NotEmpty(c.GetCveBaseInfo().GetCve())
		s.True(c.GetPreferredCvss() > 0)
	}
}

// TestSACDeniedAccess verifies that SAC-denied contexts cannot access CVEs.
func (s *vmCVEV2DataStoreTestSuite) TestSACDeniedAccess() {
	cve := s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	_, found, err := s.datastore.Get(s.noSacCtx, cve.GetId())
	s.NoError(err)
	s.False(found)

	exists, err := s.datastore.Exists(s.noSacCtx, cve.GetId())
	s.NoError(err)
	s.False(exists)

	count, err := s.datastore.Count(s.noSacCtx, nil)
	s.NoError(err)
	s.Equal(0, count)

	results, err := s.datastore.Search(s.noSacCtx, nil)
	s.NoError(err)
	s.Empty(results)

	cves, err := s.datastore.GetBatch(s.noSacCtx, []string{cve.GetId()})
	s.NoError(err)
	s.Empty(cves)

	rawCVEs, err := s.datastore.SearchRawVMCVEs(s.noSacCtx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(rawCVEs)
}

// TestSACNamespaceScopedAccess verifies namespace-scoped SAC filtering.
func (s *vmCVEV2DataStoreTestSuite) TestSACNamespaceScopedAccess() {
	cve1 := s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	cve2 := s.insertFullChainAndCVE(testconsts.Cluster1, testconsts.NamespaceB, 1, 1)
	cve3 := s.insertFullChainAndCVE(testconsts.Cluster2, testconsts.NamespaceA, 1, 1)
	cve4 := s.insertFullChainAndCVE(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	contexts := testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.VirtualMachine)

	for name, tc := range map[string]struct {
		scopeKey    string
		expectedIDs []string
	}{
		"Unrestricted read access returns all CVEs": {
			scopeKey:    testutils.UnrestrictedReadCtx,
			expectedIDs: []string{cve1.GetId(), cve2.GetId(), cve3.GetId(), cve4.GetId()},
		},
		"Unrestricted read-write access returns all CVEs": {
			scopeKey:    testutils.UnrestrictedReadWriteCtx,
			expectedIDs: []string{cve1.GetId(), cve2.GetId(), cve3.GetId(), cve4.GetId()},
		},
		"Cluster1 read-write access returns only cluster1 CVEs": {
			scopeKey:    testutils.Cluster1ReadWriteCtx,
			expectedIDs: []string{cve1.GetId(), cve2.GetId()},
		},
		"Cluster2 read-write access returns only cluster2 CVEs": {
			scopeKey:    testutils.Cluster2ReadWriteCtx,
			expectedIDs: []string{cve3.GetId(), cve4.GetId()},
		},
		"Cluster3 read-write access returns no CVEs": {
			scopeKey:    testutils.Cluster3ReadWriteCtx,
			expectedIDs: nil,
		},
		"Cluster1 NamespaceA access returns only cluster1 namespaceA CVEs": {
			scopeKey:    testutils.Cluster1NamespaceAReadWriteCtx,
			expectedIDs: []string{cve1.GetId()},
		},
		"Cluster1 NamespaceB access returns only cluster1 namespaceB CVEs": {
			scopeKey:    testutils.Cluster1NamespaceBReadWriteCtx,
			expectedIDs: []string{cve2.GetId()},
		},
		"Cluster1 NamespacesAB access returns cluster1 namespaces A and B CVEs": {
			scopeKey:    testutils.Cluster1NamespacesABReadWriteCtx,
			expectedIDs: []string{cve1.GetId(), cve2.GetId()},
		},
		"Cluster2 NamespaceB access returns only cluster2 namespaceB CVEs": {
			scopeKey:    testutils.Cluster2NamespaceBReadWriteCtx,
			expectedIDs: []string{cve4.GetId()},
		},
		"Mixed cluster and namespace access returns CVEs from allowed scopes": {
			scopeKey:    testutils.MixedClusterAndNamespaceReadCtx,
			expectedIDs: []string{cve1.GetId(), cve3.GetId(), cve4.GetId()},
		},
	} {
		s.Run(name, func() {
			ctx := contexts[tc.scopeKey]

			// Test Count
			count, err := s.datastore.Count(ctx, nil)
			s.NoError(err)
			s.Equal(len(tc.expectedIDs), count)

			// Test Search
			results, err := s.datastore.Search(ctx, nil)
			s.NoError(err)
			fetchedIDs := make([]string, len(results))
			for i, r := range results {
				fetchedIDs[i] = r.ID
			}
			s.ElementsMatch(tc.expectedIDs, fetchedIDs)

			// Test Exists for a CVE in cluster1 namespaceA
			exists, err := s.datastore.Exists(ctx, cve1.GetId())
			s.NoError(err)
			expectedExists := false
			for _, id := range tc.expectedIDs {
				if id == cve1.GetId() {
					expectedExists = true
					break
				}
			}
			s.Equal(expectedExists, exists)

			// Test Get for a CVE in cluster1 namespaceA
			_, found, err := s.datastore.Get(ctx, cve1.GetId())
			s.NoError(err)
			s.Equal(expectedExists, found)

			// Test GetBatch
			allIDs := []string{cve1.GetId(), cve2.GetId(), cve3.GetId(), cve4.GetId()}
			cves, err := s.datastore.GetBatch(ctx, allIDs)
			s.NoError(err)
			batchIDs := make([]string, len(cves))
			for i, c := range cves {
				batchIDs[i] = c.GetId()
			}
			s.ElementsMatch(tc.expectedIDs, batchIDs)

			// Test SearchRawVMCVEs
			rawCVEs, err := s.datastore.SearchRawVMCVEs(ctx, search.EmptyQuery())
			s.NoError(err)
			rawIDs := make([]string, len(rawCVEs))
			for i, c := range rawCVEs {
				rawIDs[i] = c.GetId()
			}
			s.ElementsMatch(tc.expectedIDs, rawIDs)
		})
	}
}

// TestSingletonGatedByFeatureFlag verifies the singleton is nil when the feature flag is disabled.
func (s *vmCVEV2DataStoreTestSuite) TestSingletonGatedByFeatureFlag() {
	s.T().Setenv(features.VirtualMachinesEnhancedDataModel.EnvVar(), "false")
	s.Nil(Singleton())
}
