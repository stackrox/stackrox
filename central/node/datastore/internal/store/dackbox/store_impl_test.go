package dackbox

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	cveStore "github.com/stackrox/stackrox/central/cve/store"
	cveDackBoxStore "github.com/stackrox/stackrox/central/cve/store/dackbox"
	"github.com/stackrox/stackrox/central/node/datastore/internal/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestNodeStore(t *testing.T) {
	suite.Run(t, new(NodeStoreTestSuite))
}

type NodeStoreTestSuite struct {
	suite.Suite

	db    *rocksdb.RocksDB
	dacky *dackbox.DackBox

	store      store.Store
	cveStorage cveStore.Store
}

func (suite *NodeStoreTestSuite) SetupSuite() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())

	var err error
	suite.dacky, err = dackbox.NewRocksDBDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNow("failed to create counter", err.Error())
	}
	suite.store = New(suite.dacky, concurrency.NewKeyFence(), false)
	suite.cveStorage = cveDackBoxStore.New(suite.dacky, concurrency.NewKeyFence())
}

func (suite *NodeStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *NodeStoreTestSuite) TestNodes() {
	ctx := sac.WithAllAccess(context.Background())

	nodes := []*storage.Node{
		{
			Id:         "id1",
			Name:       "name1",
			K8SUpdated: types.TimestampNow(),
			Scan: &storage.NodeScan{
				ScanTime: types.TimestampNow(),
				Components: []*storage.EmbeddedNodeScanComponent{
					{
						Name:    "comp1",
						Version: "ver1",
						Vulns:   []*storage.EmbeddedVulnerability{},
					},
					{
						Name:    "comp1",
						Version: "ver2",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:               "cve1",
								VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							},
							{
								Cve:               "cve2",
								VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "ver3",
								},
							},
						},
					},
					{
						Name:    "comp2",
						Version: "ver1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:               "cve1",
								VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "ver2",
								},
							},
							{
								Cve:               "cve2",
								VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							},
						},
					},
				},
			},
			RiskScore: 30,
		},
		{
			Id:         "id2",
			Name:       "name2",
			K8SUpdated: types.TimestampNow(),
		},
	}

	// Test Add
	for _, d := range nodes {
		suite.NoError(suite.store.Upsert(ctx, d))
	}

	for _, d := range nodes {
		got, exists, err := suite.store.Get(ctx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		// Upsert sets `createdAt` for every CVE that doesn't already exist in the store, which should be same as (*storage.Node).LastUpdated.
		for _, component := range d.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				vuln.FirstSystemOccurrence = got.GetLastUpdated()
				vuln.VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
				vuln.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_NODE_VULNERABILITY}
			}
		}
		suite.Equal(d, got)
	}

	// Check that the CVEs were written with the correct timestamp.
	vuln, _, err := suite.cveStorage.Get(ctx, "cve1")
	suite.NoError(err)
	suite.Equal(nodes[0].GetLastUpdated(), vuln.GetCreatedAt())
	vuln, _, err = suite.cveStorage.Get(ctx, "cve2")
	suite.NoError(err)
	suite.Equal(nodes[0].GetLastUpdated(), vuln.GetCreatedAt())

	// Test Update
	for _, d := range nodes {
		d.Name += "1"
	}

	for _, d := range nodes {
		suite.NoError(suite.store.Upsert(ctx, d))
	}

	for _, d := range nodes {
		got, exists, err := suite.store.Get(ctx, d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(d, got)
	}

	// Test Count
	count, err := suite.store.Count(ctx)
	suite.NoError(err)
	suite.Equal(len(nodes), count)

	// Test no components and cve update, only node bucket update
	cloned := nodes[0].Clone()
	cloned.Scan.ScanTime.Seconds = cloned.Scan.ScanTime.Seconds - 500
	cloned.Name = "newname"
	cloned.Scan.Components = nil
	cloned.RiskScore = 100
	suite.NoError(suite.store.Upsert(ctx, cloned))
	got, exists, err := suite.store.Get(ctx, cloned.GetId())
	suite.NoError(err)
	suite.True(exists)
	// Since the scan is not outdated, node update does not go through.
	suite.Equal("newname", got.GetName())
	// The node in store should still have components since it has fresher scan.
	suite.Len(got.GetScan().GetComponents(), len(nodes[0].GetScan().GetComponents()))
	// Risk score of stored node should be picked up.
	suite.Equal(nodes[0].GetRiskScore(), got.GetRiskScore())

	// Since nodes[0] is updated in store, update the "expected" object
	nodes[0].LastUpdated = got.GetLastUpdated()
	nodes[0].Name = "newname"

	// Test first node occurrence of CVE that is already discovered in system.
	nodes[1].Scan = &storage.NodeScan{
		Components: []*storage.EmbeddedNodeScanComponent{
			{
				Name:    "comp1",
				Version: "ver1",
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve:               "cve1",
						VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
					},
				},
			},
		},
	}

	suite.NoError(suite.store.Upsert(ctx, nodes[1]))

	got, exists, err = suite.store.Get(ctx, nodes[1].GetId())
	suite.NoError(err)
	suite.True(exists)
	nodes[1].GetScan().GetComponents()[0].GetVulns()[0].FirstSystemOccurrence = nodes[0].GetScan().GetComponents()[1].GetVulns()[0].FirstSystemOccurrence
	nodes[1].GetScan().GetComponents()[0].GetVulns()[0].VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
	nodes[1].GetScan().GetComponents()[0].GetVulns()[0].VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_NODE_VULNERABILITY}
	suite.Equal(nodes[1], got)

	// Test second occurrence of a CVE in a node
	nodes[0].GetScan().GetComponents()[0].Vulns = append(nodes[0].GetScan().GetComponents()[0].Vulns,
		&storage.EmbeddedVulnerability{
			Cve:               "cve1",
			VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
		})

	suite.NoError(suite.store.Upsert(ctx, nodes[0]))

	got, exists, err = suite.store.Get(ctx, nodes[0].GetId())
	suite.NoError(err)
	suite.True(exists)
	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].FirstSystemOccurrence = nodes[0].GetScan().GetComponents()[1].GetVulns()[0].FirstSystemOccurrence
	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_NODE_VULNERABILITY}
	suite.Equal(nodes[0], got)

	// Test Delete
	for _, d := range nodes {
		err := suite.store.Delete(ctx, d.GetId())
		suite.NoError(err)
	}

	// Test Count
	count, err = suite.store.Count(ctx)
	suite.NoError(err)
	suite.Equal(0, count)

	// Check that the CVEs are removed.
	count, err = suite.cveStorage.Count(ctx)
	suite.NoError(err)
	suite.Equal(0, count)
}

func (suite *NodeStoreTestSuite) TestNodeUpsert() {
	ctx := sac.WithAllAccess(context.Background())
	node := &storage.Node{
		Id:         "id1",
		Name:       "name1",
		K8SUpdated: types.TimestampNow(),
		Scan: &storage.NodeScan{
			ScanTime: types.TimestampNow(),
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "comp1",
					Version: "ver1",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
			},
		},
		RiskScore: 30,
	}

	suite.NoError(suite.store.Upsert(ctx, node))
	storedNode, exists, err := suite.store.Get(ctx, node.GetId())
	suite.NoError(err)
	suite.True(exists)

	// Update node (non-scan update).
	node = storedNode.Clone()
	newNode := storedNode.Clone()
	newNode.Annotations = map[string]string{
		"hi": "bye",
	}
	newNode.K8SUpdated = types.TimestampNow()

	expectedNode := newNode.Clone()

	suite.NoError(suite.store.Upsert(ctx, newNode))
	storedNode, exists, err = suite.store.Get(ctx, newNode.GetId())
	suite.NoError(err)
	suite.True(exists)
	suite.True(expectedNode.GetLastUpdated().Compare(storedNode.GetLastUpdated()) < 0)
	expectedNode.LastUpdated = storedNode.GetLastUpdated()
	suite.Equal(expectedNode, storedNode)

	// "Asynchronously" update scan with old node data.
	node.Scan.ScanTime = types.TimestampNow()
	expectedNode.Scan.ScanTime = node.GetScan().GetScanTime()

	suite.NoError(suite.store.Upsert(ctx, node))
	storedNode, exists, err = suite.store.Get(ctx, node.GetId())
	suite.NoError(err)
	suite.True(exists)
	suite.True(expectedNode.GetLastUpdated().Compare(storedNode.GetLastUpdated()) < 0)
	expectedNode.LastUpdated = storedNode.GetLastUpdated()
	suite.Equal(expectedNode, storedNode)

	suite.NoError(suite.store.Delete(ctx, node.GetId()))
}
