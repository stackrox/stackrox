//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	nodeCveStore "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type NodesStoreSuite struct {
	suite.Suite
	ctx  context.Context
	pool postgres.DB
}

func TestNodesStore(t *testing.T) {
	suite.Run(t, new(NodesStoreSuite))
}

func (s *NodesStoreSuite) SetupTest() {

	s.ctx = sac.WithAllAccess(context.Background())
	s.pool = pgtest.ForT(s.T())
}

func (s *NodesStoreSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *NodesStoreSuite) TestStore() {
	store := New(s.pool, false, concurrency.NewKeyFence())

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range node.GetScan().GetComponents() {
		comp.Vulns = nil
	}

	foundNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)

	enricher.FillScanStats(node)
	s.NoError(store.Upsert(s.ctx, node))
	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	cloned := node.CloneVT()

	for _, component := range cloned.GetScan().GetComponents() {
		for _, vuln := range component.GetVulnerabilities() {
			vuln.CveBaseInfo.CreatedAt = node.GetLastUpdated()
		}
	}
	protoassert.Equal(s.T(), cloned, foundNode)

	nodeCount, err := store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(nodeCount, 1)

	nodeExists, err := store.Exists(s.ctx, node.GetId())
	s.NoError(err)
	s.True(nodeExists)
	s.NoError(store.Upsert(s.ctx, node))

	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)

	// Reconcile the timestamps that are set during upsert.
	cloned.LastUpdated = foundNode.GetLastUpdated()
	protoassert.Equal(s.T(), cloned, foundNode)

	s.NoError(store.Delete(s.ctx, node.GetId()))
	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)
}

func (s *NodesStoreSuite) TestWalkByQuery() {
	store := New(s.pool, false, concurrency.NewKeyFence())

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	node2 := node.CloneVT()
	node2.Id = uuid.NewDummy().String()

	s.NoError(store.Upsert(s.ctx, node))
	s.NoError(store.Upsert(s.ctx, node2))

	walkFn := func(obj *storage.Node) error {
		if obj.GetId() != node.GetId() {
			return fmt.Errorf("expected node1 but got %s", obj.GetId())
		}
		return nil
	}

	q := search.NewQueryBuilder().AddExactMatches(search.NodeID, node.GetId()).ProtoQuery()
	s.NoError(store.WalkByQuery(s.ctx, q, walkFn))
}

func (s *NodesStoreSuite) TestStore_UpsertWithoutScan() {
	store := New(s.pool, false, concurrency.NewKeyFence())

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	foundNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)

	s.NoError(store.Upsert(s.ctx, node))

	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotNil(foundNode.GetScan())

	node = foundNode.CloneVT()
	node.Scan = nil
	s.NoError(store.Upsert(s.ctx, node))

	newNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)

	// We expect only LastUpdated to have changed.
	foundNode.LastUpdated = newNode.GetLastUpdated()
	protoassert.Equal(s.T(), foundNode, newNode)
}

func (s *NodesStoreSuite) TestStore_OrphanedCVEs() {
	s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "true")
	if !env.OrphanedCVEsKeepAlive.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_ORPHANED_CVES_KEEP_ALIVE disabled")
		s.T().SkipNow()
	}
	defer s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "false")

	store := New(s.pool, false, concurrency.NewKeyFence())

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	foundNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)

	s.NoError(store.Upsert(s.ctx, node))

	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotEmpty(foundNode.GetScan().GetComponents())
	s.NotEmpty(foundNode.GetScan().GetComponents()[0].GetVulnerabilities())

	prevVulns := foundNode.GetScan().GetComponents()[0].GetVulnerabilities()
	vulnNames := set.NewStringSet()
	for _, cve := range prevVulns {
		vulnNames.Add(cve.GetCveBaseInfo().GetCve())
	}

	// Remove all node Vulnerabilities
	node = foundNode.CloneVT()
	node.GetScan().GetComponents()[0].Vulnerabilities = nil
	iTime := time.Now()
	node.Scan.ScanTime = protocompat.ConvertTimeToTimestampOrNil(&iTime)
	s.NoError(store.Upsert(s.ctx, node))

	// Updated node does not contain any CVEs
	newNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotEmpty(newNode.GetScan().GetComponents())
	s.Empty(newNode.GetScan().GetComponents()[0].GetVulnerabilities())

	// Removed vulns should be marked orphaned in node_cves table
	cveStore := nodeCveStore.New(s.pool)
	orphanedCVEs, err := cveStore.GetByQuery(s.ctx, search.NewQueryBuilder().AddBools(search.CVEOrphaned, true).ProtoQuery())
	s.NoError(err)
	s.NotEmpty(orphanedCVEs)
	for _, cve := range orphanedCVEs {
		s.NotNil(cve.OrphanedTime)
		s.True(vulnNames.Contains(cve.GetCveBaseInfo().GetCve()))
	}

	orphanedCveIDToCve := make(map[string]*storage.NodeCVE)
	for _, cve := range orphanedCVEs {
		orphanedCveIDToCve[cve.GetId()] = cve
	}

	// Add back prev removed vulnerabilities
	newNode.GetScan().GetComponents()[0].Vulnerabilities = prevVulns
	iTime = time.Now()
	newNode.Scan.ScanTime = protocompat.ConvertTimeToTimestampOrNil(&iTime)
	enricher.FillScanStats(newNode)
	s.NoError(store.Upsert(s.ctx, newNode))

	// Vulns are added back to the node
	foundNode, exists, err = store.Get(s.ctx, newNode.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotEmpty(newNode.GetScan().GetComponents())
	s.NotEmpty(newNode.GetScan().GetComponents()[0].GetVulnerabilities())

	// CVEs should no longer be marked orphaned
	nodeCVEs, err := cveStore.GetByQuery(s.ctx, search.NewQueryBuilder().AddExactMatches(search.NodeID, foundNode.GetId()).ProtoQuery())
	s.NoError(err)
	s.NotEmpty(nodeCVEs)
	for _, cve := range nodeCVEs {
		s.False(cve.Orphaned)
		s.Nil(cve.OrphanedTime)
		val, ok := orphanedCveIDToCve[cve.GetId()]
		s.True(ok)
		s.Equal(val.GetCveBaseInfo().GetCreatedAt(), cve.GetCveBaseInfo().GetCreatedAt())
	}

	metadatas, missing, err := store.GetManyNodeMetadata(s.ctx, []string{newNode.GetId(), uuid.NewDummy().String()})
	s.NoError(err)
	s.Equal(missing, []int{1})
	protoassert.SlicesEqual(s.T(), []*storage.Node{stripComponents(newNode)}, metadatas)
}

func stripComponents(n *storage.Node) *storage.Node {
	node := n.CloneVT()
	node.GetScan().Components = nil
	return node
}

func (s *NodesStoreSuite) TestGetWithTransactionContext() {
	store := New(s.pool, false, concurrency.NewKeyFence())

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range node.GetScan().GetComponents() {
		comp.Vulns = nil
	}

	// Insert test data
	s.NoError(store.Upsert(s.ctx, node))

	// Create explicit transaction
	tx, err := s.pool.Begin(s.ctx)
	s.NoError(err)

	// Pass transaction context to Get
	ctx := postgres.ContextWithTx(s.ctx, tx)
	retrieved, ok, err := store.Get(ctx, node.GetId())

	s.NoError(err)
	s.True(ok)
	s.Equal(node.GetId(), retrieved.GetId())
	s.NoError(tx.Rollback(s.ctx))
}

func (s *NodesStoreSuite) TestGetManyWithTransactionContext() {
	store := New(s.pool, false, concurrency.NewKeyFence())

	node1 := &storage.Node{}
	s.NoError(testutils.FullInit(node1, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range node1.GetScan().GetComponents() {
		comp.Vulns = nil
	}

	node2 := &storage.Node{}
	s.NoError(testutils.FullInit(node2, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range node2.GetScan().GetComponents() {
		comp.Vulns = nil
	}

	// Insert test data
	s.NoError(store.Upsert(s.ctx, node1))
	s.NoError(store.Upsert(s.ctx, node2))

	// Create explicit transaction
	tx, err := s.pool.Begin(s.ctx)
	s.NoError(err)

	// Pass transaction context to GetMany
	ctx := postgres.ContextWithTx(s.ctx, tx)
	nodes, missing, err := store.GetMany(ctx, []string{node1.GetId(), node2.GetId()})

	s.NoError(err)
	s.Empty(missing)
	s.Len(nodes, 2)
	s.NoError(tx.Rollback(s.ctx))
}

func (s *NodesStoreSuite) TestWalkByQueryWithTransactionContext() {
	store := New(s.pool, false, concurrency.NewKeyFence())

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range node.GetScan().GetComponents() {
		comp.Vulns = nil
	}

	// Insert test data
	s.NoError(store.Upsert(s.ctx, node))

	// Create explicit transaction
	tx, err := s.pool.Begin(s.ctx)
	s.NoError(err)

	// Pass transaction context to WalkByQuery
	ctx := postgres.ContextWithTx(s.ctx, tx)
	var count int
	walkFn := func(n *storage.Node) error {
		count++
		s.Equal(node.GetId(), n.GetId())
		return nil
	}

	q := search.NewQueryBuilder().AddExactMatches(search.NodeID, node.GetId()).ProtoQuery()
	s.NoError(store.WalkByQuery(ctx, q, walkFn))
	s.Equal(1, count)
	s.NoError(tx.Rollback(s.ctx))
}

// TestStore_DenormalizedCountsUpdatedAfterOrphaning tests that the denormalized CVE counts
// in the nodes table are recalculated after CVEs are marked as orphaned.
func (s *NodesStoreSuite) TestStore_DenormalizedCountsUpdatedAfterOrphaning() {
	s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "true")
	if !env.OrphanedCVEsKeepAlive.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_ORPHANED_CVES_KEEP_ALIVE disabled")
		s.T().SkipNow()
	}
	defer s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "false")

	store := New(s.pool, false, concurrency.NewKeyFence())

	// Create a node with multiple CVEs
	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	// Ensure the node has components with vulnerabilities
	s.Require().NotEmpty(node.GetScan().GetComponents())
	s.Require().NotEmpty(node.GetScan().GetComponents()[0].GetVulnerabilities())

	// Count initial CVEs and fixable CVEs
	initialCVECount := int32(0)
	cveSet := make(map[string]bool)
	for _, comp := range node.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulnerabilities() {
			cveName := vuln.GetCveBaseInfo().GetCve()
			if _, exists := cveSet[cveName]; !exists {
				cveSet[cveName] = vuln.GetFixedBy() != ""
				initialCVECount++
			}
		}
	}

	s.Require().Greater(initialCVECount, int32(0), "Test requires node with CVEs")

	// Fill scan stats to populate denormalized counts
	enricher.FillScanStats(node)

	// Upsert the node
	s.NoError(store.Upsert(s.ctx, node))

	// Verify initial counts match
	foundNode, exists, err := store.GetNodeMetadata(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(initialCVECount, foundNode.GetCves(), "Initial CVE count should match")

	// Update the node to remove all vulnerabilities (simulating Scanner V4 replacing Scanner V2)
	updatedNode := foundNode.CloneVT()
	newScanTime := time.Now().Add(time.Minute)
	updatedNode.Scan = &storage.NodeScan{
		ScanTime:   protocompat.ConvertTimeToTimestampOrNil(&newScanTime),
		Components: []*storage.EmbeddedNodeScanComponent{}, // No components = no CVEs
	}

	// Fill scan stats for the updated node (will set counts to 0)
	enricher.FillScanStats(updatedNode)

	s.NoError(store.Upsert(s.ctx, updatedNode))

	// Verify the denormalized counts are updated to 0
	finalNode, exists, err := store.GetNodeMetadata(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(int32(0), finalNode.GetCves(), "CVE count should be 0 after all CVEs are orphaned")
	s.Equal(int32(0), finalNode.GetFixableCves(), "Fixable CVE count should be 0 after all CVEs are orphaned")
}

// TestStore_DenormalizedCountsPartialOrphaning tests that counts are correctly updated
// when only some CVEs are orphaned.
func (s *NodesStoreSuite) TestStore_DenormalizedCountsPartialOrphaning() {
	s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "true")
	if !env.OrphanedCVEsKeepAlive.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_ORPHANED_CVES_KEEP_ALIVE disabled")
		s.T().SkipNow()
	}
	defer s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "false")

	store := New(s.pool, false, concurrency.NewKeyFence())

	// Create a node with multiple components and CVEs
	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	// Ensure we have at least 1 component with vulnerabilities
	s.Require().NotEmpty(node.GetScan().GetComponents())
	s.Require().NotEmpty(node.GetScan().GetComponents()[0].GetVulnerabilities())

	// If there's only one component, duplicate it to ensure we have at least 2
	if len(node.GetScan().GetComponents()) < 2 {
		// Clone the first component and give it a different name/version to create unique CVEs
		secondComp := node.GetScan().GetComponents()[0].CloneVT()
		secondComp.Name = secondComp.GetName() + "-second"
		secondComp.Version = secondComp.GetVersion() + "-v2"
		// Modify CVE names to make them unique
		for _, vuln := range secondComp.GetVulnerabilities() {
			vuln.CveBaseInfo.Cve = vuln.GetCveBaseInfo().GetCve() + "-copy"
		}
		node.GetScan().Components = append(node.GetScan().GetComponents(), secondComp)
	}

	s.Require().Greater(len(node.GetScan().GetComponents()), 1, "Test requires at least 2 components")

	// Fill scan stats to populate denormalized counts
	enricher.FillScanStats(node)

	// Upsert initial node
	s.NoError(store.Upsert(s.ctx, node))

	foundNode, exists, err := store.GetNodeMetadata(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	initialCVECount := foundNode.GetCves()

	s.Require().Greater(initialCVECount, int32(0))

	// Remove only the first component (partial orphaning)
	updatedNode, _, _ := store.Get(s.ctx, node.GetId())
	updatedNode.GetScan().Components = updatedNode.GetScan().GetComponents()[1:] // Keep all but first
	newScanTime := time.Now()
	updatedNode.Scan.ScanTime = protocompat.ConvertTimeToTimestampOrNil(&newScanTime)

	// Fill scan stats for the updated node
	enricher.FillScanStats(updatedNode)

	// Count expected remaining CVEs
	expectedCVECount := int32(0)
	expectedFixableCount := int32(0)
	cveSet := make(map[string]bool)
	for _, comp := range updatedNode.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulnerabilities() {
			cveName := vuln.GetCveBaseInfo().GetCve()
			if _, exists := cveSet[cveName]; !exists {
				cveSet[cveName] = true
				expectedCVECount++
				if vuln.GetFixedBy() != "" {
					expectedFixableCount++
				}
			}
		}
	}

	s.NoError(store.Upsert(s.ctx, updatedNode))

	// Verify counts reflect only remaining CVEs
	finalNode, exists, err := store.GetNodeMetadata(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(expectedCVECount, finalNode.GetCves(), "CVE count should match remaining CVEs")
	s.Equal(expectedFixableCount, finalNode.GetFixableCves(), "Fixable CVE count should match remaining fixable CVEs")
	s.Less(finalNode.GetCves(), initialCVECount, "CVE count should be less than initial")
}

// TestStore_DenormalizedCountsMultipleNodes tests that updating one node's counts
// doesn't affect other nodes.
func (s *NodesStoreSuite) TestStore_DenormalizedCountsMultipleNodes() {
	s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "true")
	if !env.OrphanedCVEsKeepAlive.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_ORPHANED_CVES_KEEP_ALIVE disabled")
		s.T().SkipNow()
	}
	defer s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "false")

	store := New(s.pool, false, concurrency.NewKeyFence())

	// Create two nodes
	node1 := &storage.Node{}
	s.NoError(testutils.FullInit(node1, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	node2 := &storage.Node{}
	s.NoError(testutils.FullInit(node2, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	// Fill scan stats to populate denormalized counts
	enricher.FillScanStats(node1)
	enricher.FillScanStats(node2)

	s.NoError(store.Upsert(s.ctx, node1))
	s.NoError(store.Upsert(s.ctx, node2))

	// Get initial counts for both nodes
	foundNode1, _, _ := store.GetNodeMetadata(s.ctx, node1.GetId())
	foundNode2, _, _ := store.GetNodeMetadata(s.ctx, node2.GetId())

	node1InitialCVEs := foundNode1.GetCves()
	node2InitialCVEs := foundNode2.GetCves()

	s.Require().Greater(node1InitialCVEs, int32(0))
	s.Require().Greater(node2InitialCVEs, int32(0))

	// Orphan all CVEs from node1 only
	updatedNode1, _, _ := store.Get(s.ctx, node1.GetId())
	newScanTime := time.Now()
	updatedNode1.Scan = &storage.NodeScan{
		ScanTime:   protocompat.ConvertTimeToTimestampOrNil(&newScanTime),
		Components: []*storage.EmbeddedNodeScanComponent{},
	}

	// Fill scan stats for the updated node (will set counts to 0)
	enricher.FillScanStats(updatedNode1)

	s.NoError(store.Upsert(s.ctx, updatedNode1))

	// Verify node1 has 0 CVEs
	finalNode1, _, _ := store.GetNodeMetadata(s.ctx, node1.GetId())
	s.Equal(int32(0), finalNode1.GetCves(), "Node1 should have 0 CVEs")

	// Verify node2 is unchanged
	finalNode2, _, _ := store.GetNodeMetadata(s.ctx, node2.GetId())
	s.Equal(node2InitialCVEs, finalNode2.GetCves(), "Node2 CVE count should be unchanged")
}

// TestStore_OrphanedComponentCVEEdgesCleanup tests that orphaned component-CVE edges
// are cleaned up, which then allows CVEs to be properly orphaned.
// This simulates Scanner V2 -> V4 migration where V4 sends 0 components.
func (s *NodesStoreSuite) TestStore_OrphanedComponentCVEEdgesCleanup() {
	store := New(s.pool, false, concurrency.NewKeyFence())

	// Create a node with components and CVEs (simulating Scanner V2)
	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	s.Require().NotEmpty(node.GetScan().GetComponents())
	s.Require().NotEmpty(node.GetScan().GetComponents()[0].GetVulnerabilities())

	enricher.FillScanStats(node)
	initialCVECount := node.GetCves()
	s.Require().Greater(initialCVECount, int32(0))

	s.NoError(store.Upsert(s.ctx, node))

	// Verify initial state
	foundNode, _, _ := store.GetNodeMetadata(s.ctx, node.GetId())
	s.Equal(initialCVECount, foundNode.GetCves())

	// Update node with 0 components (simulating Scanner V4)
	updatedNode := foundNode.CloneVT()
	newScanTime := time.Now()
	updatedNode.Scan = &storage.NodeScan{
		ScanTime:   protocompat.ConvertTimeToTimestampOrNil(&newScanTime),
		Components: []*storage.EmbeddedNodeScanComponent{}, // Scanner V4 sends no components
	}
	enricher.FillScanStats(updatedNode)

	s.NoError(store.Upsert(s.ctx, updatedNode))

	// Verify CVE count is now 0
	finalNode, _, _ := store.GetNodeMetadata(s.ctx, node.GetId())
	s.Equal(int32(0), finalNode.GetCves(), "CVE count should be 0 after components removed")

	// Verify orphaned component-CVE edges were cleaned up
	cveStore := nodeCveStore.New(s.pool)
	remainingCVEs, err := cveStore.GetByQuery(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Empty(remainingCVEs, "All CVEs should be deleted when component edges are orphaned")

	// Directly verify component-CVE edges were CASCADE deleted
	var edgeCount int
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_components_cves_edges").Scan(&edgeCount)
	s.NoError(err)
	s.Equal(0, edgeCount, "Component-CVE edges should be CASCADE deleted when components are deleted")

	// Verify components themselves were deleted
	var componentCount int
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_components").Scan(&componentCount)
	s.NoError(err)
	s.Equal(0, componentCount, "Components should be deleted when orphaned")
}

// TestStore_SharedComponentsNotDeletedWhenStillReferenced tests that when multiple nodes
// share the same component (same name#version#os composite ID), rescanning one node to
// remove that component doesn't delete the component or its CVE edges if the other node
// still references it.
func (s *NodesStoreSuite) TestStore_SharedComponentsNotDeletedWhenStillReferenced() {
	store := New(s.pool, false, concurrency.NewKeyFence())

	// Create node1 with a full init to get all required fields
	node1 := &storage.Node{}
	s.NoError(testutils.FullInit(node1, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	s.Require().NotEmpty(node1.GetScan().GetComponents())

	// Create node2 and give it the SAME component as node1 (to test shared component handling)
	node2 := &storage.Node{}
	s.NoError(testutils.FullInit(node2, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	// Make node2 share the same component as node1 by copying it
	// The component ID is composite: name#version#os, so copying the component and using same OS creates a shared component
	sharedComponent := node1.GetScan().GetComponents()[0].CloneVT()
	node2.Scan.Components = []*storage.EmbeddedNodeScanComponent{sharedComponent}
	node2.Scan.OperatingSystem = node1.GetScan().GetOperatingSystem()

	enricher.FillScanStats(node1)
	enricher.FillScanStats(node2)

	s.NoError(store.Upsert(s.ctx, node1))
	s.NoError(store.Upsert(s.ctx, node2))

	// Verify both nodes have the component
	var node1EdgeCount, node2EdgeCount int
	err := s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_component_edges WHERE nodeid = $1::uuid", node1.GetId()).Scan(&node1EdgeCount)
	s.NoError(err)
	s.Equal(1, node1EdgeCount, "Node1 should have 1 component edge")

	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_component_edges WHERE nodeid = $1::uuid", node2.GetId()).Scan(&node2EdgeCount)
	s.NoError(err)
	s.Equal(1, node2EdgeCount, "Node2 should have 1 component edge")

	// Verify there's only 1 shared component (not 2 duplicates)
	var totalComponents int
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_components").Scan(&totalComponents)
	s.NoError(err)
	s.Equal(1, totalComponents, "Should only have 1 shared component, not duplicates")

	// Verify the component-CVE edge exists
	var componentCVEEdgeCount int
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_components_cves_edges").Scan(&componentCVEEdgeCount)
	s.NoError(err)
	s.Equal(1, componentCVEEdgeCount, "Should have 1 component-CVE edge for the shared component")

	// Now rescan node1 with 0 components (simulating Scanner V4 or component removal)
	updatedNode1, _, _ := store.Get(s.ctx, node1.GetId())
	newScanTime := time.Now()
	updatedNode1.Scan = &storage.NodeScan{
		ScanTime:   protocompat.ConvertTimeToTimestampOrNil(&newScanTime),
		Components: []*storage.EmbeddedNodeScanComponent{}, // No components
	}
	enricher.FillScanStats(updatedNode1)

	s.NoError(store.Upsert(s.ctx, updatedNode1))

	// Verify node1's edge was removed
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_component_edges WHERE nodeid = $1::uuid", node1.GetId()).Scan(&node1EdgeCount)
	s.NoError(err)
	s.Equal(0, node1EdgeCount, "Node1 should have 0 component edges after rescan")

	// CRITICAL: Verify node2's edge still exists
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_component_edges WHERE nodeid = $1::uuid", node2.GetId()).Scan(&node2EdgeCount)
	s.NoError(err)
	s.Equal(1, node2EdgeCount, "Node2 should still have its component edge")

	// CRITICAL: Verify the shared component was NOT deleted (node2 still references it)
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_components").Scan(&totalComponents)
	s.NoError(err)
	s.Equal(1, totalComponents, "Shared component should NOT be deleted while node2 still references it")

	// CRITICAL: Verify the component-CVE edge still exists (component wasn't deleted)
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_components_cves_edges").Scan(&componentCVEEdgeCount)
	s.NoError(err)
	s.Equal(1, componentCVEEdgeCount, "Component-CVE edge should still exist for the shared component")

	// Verify node2's CVE count is still correct
	finalNode2, _, _ := store.GetNodeMetadata(s.ctx, node2.GetId())
	s.Equal(int32(1), finalNode2.GetCves(), "Node2 should still have 1 CVE")

	// Now rescan node2 with 0 components - NOW the component should be deleted
	updatedNode2, _, _ := store.Get(s.ctx, node2.GetId())
	updatedNode2.Scan = &storage.NodeScan{
		ScanTime:   protocompat.ConvertTimeToTimestampOrNil(&newScanTime),
		Components: []*storage.EmbeddedNodeScanComponent{},
	}
	enricher.FillScanStats(updatedNode2)

	s.NoError(store.Upsert(s.ctx, updatedNode2))

	// NOW the component should be deleted (no nodes reference it)
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_components").Scan(&totalComponents)
	s.NoError(err)
	s.Equal(0, totalComponents, "Component should be deleted when no nodes reference it")

	// And the component-CVE edge should be CASCADE deleted
	err = s.pool.QueryRow(s.ctx, "SELECT COUNT(*) FROM node_components_cves_edges").Scan(&componentCVEEdgeCount)
	s.NoError(err)
	s.Equal(0, componentCVEEdgeCount, "Component-CVE edge should be CASCADE deleted when component is deleted")
}

// TestStore_SerializedBlobCVECountsMatchColumns verifies that the serialized blob's CVE counts
// stay in sync with the denormalized Cves/FixableCves columns after CVEs are orphaned.
func (s *NodesStoreSuite) TestStore_SerializedBlobCVECountsMatchColumns() {
	s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "true")
	if !env.OrphanedCVEsKeepAlive.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_ORPHANED_CVES_KEEP_ALIVE disabled")
		s.T().SkipNow()
	}
	defer s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "false")

	store := New(s.pool, false, concurrency.NewKeyFence())

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	s.Require().NotEmpty(node.GetScan().GetComponents())
	s.Require().NotEmpty(node.GetScan().GetComponents()[0].GetVulnerabilities())

	enricher.FillScanStats(node)
	s.NoError(store.Upsert(s.ctx, node))

	// Remove all components so CVEs become orphaned, changing counts
	updatedNode, _, _ := store.Get(s.ctx, node.GetId())
	newScanTime := time.Now()
	updatedNode.Scan = &storage.NodeScan{
		ScanTime:   protocompat.ConvertTimeToTimestampOrNil(&newScanTime),
		Components: []*storage.EmbeddedNodeScanComponent{},
	}
	enricher.FillScanStats(updatedNode)
	s.NoError(store.Upsert(s.ctx, updatedNode))

	// Read the serialized blob and the individual columns directly
	var serializedData []byte
	var columnCves, columnFixableCves int32
	err := s.pool.QueryRow(s.ctx,
		"SELECT serialized, Cves, FixableCves FROM nodes WHERE Id = $1",
		node.GetId(),
	).Scan(&serializedData, &columnCves, &columnFixableCves)
	s.NoError(err)

	var deserialized storage.Node
	s.NoError(deserialized.UnmarshalVTUnsafe(serializedData))

	s.Equal(columnCves, deserialized.GetCves(),
		"Serialized blob CVE count must match the Cves column")
	s.Equal(columnFixableCves, deserialized.GetFixableCves(),
		"Serialized blob fixable CVE count must match the FixableCves column")
}
