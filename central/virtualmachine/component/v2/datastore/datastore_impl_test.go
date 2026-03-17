//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	componentStore "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/store/postgres"
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

func TestVMComponentV2DataStore(t *testing.T) {
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		t.Skip("VM enhanced data model is not enabled")
	}
	suite.Run(t, new(vmComponentV2DataStoreTestSuite))
}

type vmComponentV2DataStoreTestSuite struct {
	suite.Suite

	db        *pgtest.TestPostgres
	datastore DataStore
	vmStore   vmV2Store.Store
	scanPG    scanStore.Store

	sacCtx   context.Context
	noSacCtx context.Context
}

func (s *vmComponentV2DataStoreTestSuite) SetupTest() {
	s.db = pgtest.ForT(s.T())
	store := componentStore.New(s.db)
	s.datastore = New(store)
	s.vmStore = vmV2Postgres.New(s.db, concurrency.NewKeyFence())
	s.scanPG = scanStore.New(s.db)

	s.sacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.VirtualMachine)))
	s.noSacCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.DenyAllAccessScopeChecker())
}

func (s *vmComponentV2DataStoreTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *vmComponentV2DataStoreTestSuite) createTestVM(clusterID, namespace string, index int) *storage.VirtualMachineV2 {
	vmID := uuid.NewV5FromNonUUIDs(clusterID, fmt.Sprintf("%s-%d", namespace, index)).String()
	return &storage.VirtualMachineV2{
		Id:          vmID,
		Name:        fmt.Sprintf("vm-%d", index),
		Namespace:   namespace,
		ClusterId:   clusterID,
		ClusterName: fmt.Sprintf("cluster-%s", clusterID),
	}
}

func (s *vmComponentV2DataStoreTestSuite) createTestScan(vmID string) *storage.VirtualMachineScanV2 {
	scanID := uuid.NewV5FromNonUUIDs(vmID, "scan").String()
	return &storage.VirtualMachineScanV2{
		Id:       scanID,
		VmV2Id:   vmID,
		ScanOs:   "rhel:9",
		ScanTime: timestamppb.Now(),
		TopCvss:  7.5,
	}
}

func (s *vmComponentV2DataStoreTestSuite) createTestComponent(scanID string, index int) *storage.VirtualMachineComponentV2 {
	componentID := uuid.NewV5FromNonUUIDs(scanID, fmt.Sprintf("component-%d", index)).String()
	return &storage.VirtualMachineComponentV2{
		Id:              componentID,
		VmScanId:        scanID,
		Name:            fmt.Sprintf("pkg-%d", index),
		Version:         fmt.Sprintf("%d.0.0", index),
		Source:          storage.SourceType_OS,
		OperatingSystem: "rhel:9",
	}
}

// insertVMScanAndComponent creates a VM, scan, and component, returning the component.
func (s *vmComponentV2DataStoreTestSuite) insertVMScanAndComponent(clusterID, namespace string, vmIndex, componentIndex int) *storage.VirtualMachineComponentV2 {
	ctx := sac.WithAllAccess(context.Background())
	vm := s.createTestVM(clusterID, namespace, vmIndex)
	s.Require().NoError(s.vmStore.UpsertVM(ctx, vm))
	scan := s.createTestScan(vm.GetId())
	s.Require().NoError(s.scanPG.Upsert(ctx, scan))
	component := s.createTestComponent(scan.GetId(), componentIndex)
	compPG := componentStore.New(s.db)
	s.Require().NoError(compPG.Upsert(ctx, component))
	return component
}

// TestGet verifies basic get functionality.
func (s *vmComponentV2DataStoreTestSuite) TestGet() {
	comp := s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	retrieved, found, err := s.datastore.Get(s.sacCtx, comp.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(comp.GetId(), retrieved.GetId())
	s.Equal(comp.GetName(), retrieved.GetName())
	s.Equal(comp.GetVersion(), retrieved.GetVersion())

	// Non-existent component
	_, found, err = s.datastore.Get(s.sacCtx, uuid.NewDummy().String())
	s.NoError(err)
	s.False(found)
}

// TestExists verifies existence checks.
func (s *vmComponentV2DataStoreTestSuite) TestExists() {
	comp := s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	exists, err := s.datastore.Exists(s.sacCtx, comp.GetId())
	s.NoError(err)
	s.True(exists)

	exists, err = s.datastore.Exists(s.sacCtx, uuid.NewDummy().String())
	s.NoError(err)
	s.False(exists)
}

// TestCount verifies count functionality.
func (s *vmComponentV2DataStoreTestSuite) TestCount() {
	count, err := s.datastore.Count(s.sacCtx, nil)
	s.NoError(err)
	s.Equal(0, count)

	s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 2, 1)

	count, err = s.datastore.Count(s.sacCtx, nil)
	s.NoError(err)
	s.Equal(2, count)
}

// TestGetBatch verifies batch retrieval.
func (s *vmComponentV2DataStoreTestSuite) TestGetBatch() {
	comp1 := s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	comp2 := s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 2, 1)

	components, err := s.datastore.GetBatch(s.sacCtx, []string{comp1.GetId(), comp2.GetId()})
	s.NoError(err)
	s.Len(components, 2)

	fetchedIDs := make([]string, len(components))
	for i, c := range components {
		fetchedIDs[i] = c.GetId()
	}
	s.ElementsMatch([]string{comp1.GetId(), comp2.GetId()}, fetchedIDs)
}

// TestSearch verifies search functionality.
func (s *vmComponentV2DataStoreTestSuite) TestSearch() {
	comp1 := s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	comp2 := s.insertVMScanAndComponent(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	results, err := s.datastore.Search(s.sacCtx, nil)
	s.NoError(err)
	s.Len(results, 2)

	fetchedIDs := make([]string, len(results))
	for i, r := range results {
		fetchedIDs[i] = r.ID
	}
	s.ElementsMatch([]string{comp1.GetId(), comp2.GetId()}, fetchedIDs)
}

// TestSearchRawVMComponents verifies raw search.
func (s *vmComponentV2DataStoreTestSuite) TestSearchRawVMComponents() {
	s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	s.insertVMScanAndComponent(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	components, err := s.datastore.SearchRawVMComponents(s.sacCtx, search.EmptyQuery())
	s.NoError(err)
	s.Len(components, 2)

	for _, c := range components {
		s.NotEmpty(c.GetId())
		s.NotEmpty(c.GetName())
		s.NotEmpty(c.GetVersion())
	}
}

// TestSACDeniedAccess verifies that SAC-denied contexts cannot access components.
func (s *vmComponentV2DataStoreTestSuite) TestSACDeniedAccess() {
	comp := s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)

	_, found, err := s.datastore.Get(s.noSacCtx, comp.GetId())
	s.NoError(err)
	s.False(found)

	exists, err := s.datastore.Exists(s.noSacCtx, comp.GetId())
	s.NoError(err)
	s.False(exists)

	count, err := s.datastore.Count(s.noSacCtx, nil)
	s.NoError(err)
	s.Equal(0, count)

	results, err := s.datastore.Search(s.noSacCtx, nil)
	s.NoError(err)
	s.Empty(results)

	components, err := s.datastore.GetBatch(s.noSacCtx, []string{comp.GetId()})
	s.NoError(err)
	s.Empty(components)

	rawComponents, err := s.datastore.SearchRawVMComponents(s.noSacCtx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(rawComponents)
}

// TestSACNamespaceScopedAccess verifies namespace-scoped SAC filtering.
func (s *vmComponentV2DataStoreTestSuite) TestSACNamespaceScopedAccess() {
	comp1 := s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceA, 1, 1)
	comp2 := s.insertVMScanAndComponent(testconsts.Cluster1, testconsts.NamespaceB, 1, 1)
	comp3 := s.insertVMScanAndComponent(testconsts.Cluster2, testconsts.NamespaceA, 1, 1)
	comp4 := s.insertVMScanAndComponent(testconsts.Cluster2, testconsts.NamespaceB, 1, 1)

	contexts := testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.VirtualMachine)

	for name, tc := range map[string]struct {
		scopeKey    string
		expectedIDs []string
	}{
		"Unrestricted read access returns all components": {
			scopeKey:    testutils.UnrestrictedReadCtx,
			expectedIDs: []string{comp1.GetId(), comp2.GetId(), comp3.GetId(), comp4.GetId()},
		},
		"Unrestricted read-write access returns all components": {
			scopeKey:    testutils.UnrestrictedReadWriteCtx,
			expectedIDs: []string{comp1.GetId(), comp2.GetId(), comp3.GetId(), comp4.GetId()},
		},
		"Cluster1 read-write access returns only cluster1 components": {
			scopeKey:    testutils.Cluster1ReadWriteCtx,
			expectedIDs: []string{comp1.GetId(), comp2.GetId()},
		},
		"Cluster2 read-write access returns only cluster2 components": {
			scopeKey:    testutils.Cluster2ReadWriteCtx,
			expectedIDs: []string{comp3.GetId(), comp4.GetId()},
		},
		"Cluster3 read-write access returns no components": {
			scopeKey:    testutils.Cluster3ReadWriteCtx,
			expectedIDs: nil,
		},
		"Cluster1 NamespaceA access returns only cluster1 namespaceA components": {
			scopeKey:    testutils.Cluster1NamespaceAReadWriteCtx,
			expectedIDs: []string{comp1.GetId()},
		},
		"Cluster1 NamespaceB access returns only cluster1 namespaceB components": {
			scopeKey:    testutils.Cluster1NamespaceBReadWriteCtx,
			expectedIDs: []string{comp2.GetId()},
		},
		"Cluster1 NamespacesAB access returns cluster1 namespaces A and B components": {
			scopeKey:    testutils.Cluster1NamespacesABReadWriteCtx,
			expectedIDs: []string{comp1.GetId(), comp2.GetId()},
		},
		"Cluster2 NamespaceB access returns only cluster2 namespaceB components": {
			scopeKey:    testutils.Cluster2NamespaceBReadWriteCtx,
			expectedIDs: []string{comp4.GetId()},
		},
		"Mixed cluster and namespace access returns components from allowed scopes": {
			scopeKey:    testutils.MixedClusterAndNamespaceReadCtx,
			expectedIDs: []string{comp1.GetId(), comp3.GetId(), comp4.GetId()},
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

			// Test Exists for a component in cluster1 namespaceA
			exists, err := s.datastore.Exists(ctx, comp1.GetId())
			s.NoError(err)
			expectedExists := false
			for _, id := range tc.expectedIDs {
				if id == comp1.GetId() {
					expectedExists = true
					break
				}
			}
			s.Equal(expectedExists, exists)

			// Test Get for a component in cluster1 namespaceA
			_, found, err := s.datastore.Get(ctx, comp1.GetId())
			s.NoError(err)
			s.Equal(expectedExists, found)

			// Test GetBatch
			allIDs := []string{comp1.GetId(), comp2.GetId(), comp3.GetId(), comp4.GetId()}
			components, err := s.datastore.GetBatch(ctx, allIDs)
			s.NoError(err)
			batchIDs := make([]string, len(components))
			for i, c := range components {
				batchIDs[i] = c.GetId()
			}
			s.ElementsMatch(tc.expectedIDs, batchIDs)

			// Test SearchRawVMComponents
			rawComponents, err := s.datastore.SearchRawVMComponents(ctx, search.EmptyQuery())
			s.NoError(err)
			rawIDs := make([]string, len(rawComponents))
			for i, c := range rawComponents {
				rawIDs[i] = c.GetId()
			}
			s.ElementsMatch(tc.expectedIDs, rawIDs)
		})
	}
}

// TestSingletonGatedByFeatureFlag verifies the singleton is nil when the feature flag is disabled.
func (s *vmComponentV2DataStoreTestSuite) TestSingletonGatedByFeatureFlag() {
	s.T().Setenv(features.VirtualMachinesEnhancedDataModel.EnvVar(), "false")
	s.Nil(Singleton())
}
