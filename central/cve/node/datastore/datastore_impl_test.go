//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var (
	testSuppressionQuery = searchPkg.NewQueryBuilder().AddBools(searchPkg.CVESuppressed, true).ProtoQuery()

	testAllAccessContext = sac.WithAllAccess(context.Background())
)

func TestNodeCVEDataStore(t *testing.T) {
	suite.Run(t, new(NodeCVEDataStoreSuite))
}

type NodeCVEDataStoreSuite struct {
	suite.Suite

	testDB    *pgtest.TestPostgres
	datastore DataStore
}

func (suite *NodeCVEDataStoreSuite) SetupSuite() {
	suite.testDB = pgtest.ForT(suite.T())

	ds, err := GetTestPostgresDataStore(suite.T(), suite.testDB.DB)
	suite.Require().NoError(err)
	suite.datastore = ds
}

func (suite *NodeCVEDataStoreSuite) TearDownSuite() {
	if suite.testDB != nil {
		suite.testDB.Close()
	}
}

func (suite *NodeCVEDataStoreSuite) TearDownTest() {
	// Clean up any test data after each test
	_, _ = suite.testDB.Exec(context.Background(), "TRUNCATE TABLE node_cves")
}

func getNodeWithCVEs(cves ...string) *storage.Node {
	vulns := make([]*storage.NodeVulnerability, 0, len(cves))
	for _, cve := range cves {
		vulns = append(vulns, &storage.NodeVulnerability{
			CveBaseInfo: &storage.CVEInfo{
				Cve: cve,
			},
		})
	}
	return &storage.Node{
		Scan: &storage.NodeScan{
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Vulnerabilities: vulns,
				},
			},
		},
	}
}

func (suite *NodeCVEDataStoreSuite) verifySuppressionStateNode(node *storage.Node, suppressedCVEs, unsuppressedCVEs []string) {
	cveMap := make(map[string]bool)
	for _, comp := range node.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulnerabilities() {
			cveMap[cve.ID(vuln.GetCveBaseInfo().GetCve(), node.GetScan().GetOperatingSystem())] = vuln.GetSnoozed()
		}
	}
	suite.verifySuppressionState(cveMap, suppressedCVEs, unsuppressedCVEs)
}

func (suite *NodeCVEDataStoreSuite) verifySuppressionState(cveMap map[string]bool, suppressedCVEs, unsuppressedCVEs []string) {
	for _, cve := range suppressedCVEs {
		val, ok := cveMap[cve]
		suite.True(ok)
		suite.True(val)
	}
	for _, cve := range unsuppressedCVEs {
		val, ok := cveMap[cve]
		suite.True(ok)
		suite.False(val)
	}
}

func (suite *NodeCVEDataStoreSuite) TestSuppressionCacheForNodes() {
	nodeCVEs := []*storage.NodeCVE{
		{
			Id: "CVE-ABC",
			CveBaseInfo: &storage.CVEInfo{
				Cve: "CVE-ABC",
			},
			Snoozed: true,
		},
		{
			Id: "CVE-DEF",
			CveBaseInfo: &storage.CVEInfo{
				Cve: "CVE-DEF",
			},
			Snoozed: true,
		},
	}
	// Insert real data into the datastore
	suite.NoError(suite.datastore.UpsertMany(testAllAccessContext, nodeCVEs))

	// Rebuild the cache to pick up the new data
	ds := suite.datastore.(*datastoreImpl)
	suite.NoError(ds.buildSuppressedCache())
	expectedCache := common.CVESuppressionCache{
		"CVE-ABC": {},
		"CVE-DEF": {},
	}
	suite.Equal(expectedCache, ds.cveSuppressionCache)

	// No apply these to the image
	node := getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.datastore.EnrichNodeWithSuppressedCVEs(node)
	suite.verifySuppressionStateNode(node, []string{"CVE-ABC#", "CVE-DEF#"}, []string{"CVE-GHI#"})

	start := time.Now()
	duration := 10 * time.Minute

	// Create CVE-GHI record first so it can be suppressed
	cveGHI := &storage.NodeCVE{
		Id: "CVE-GHI",
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-GHI",
		},
	}
	suite.NoError(suite.datastore.UpsertMany(testAllAccessContext, []*storage.NodeCVE{cveGHI}))

	// Clear image before suppressing
	node = getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	err := suite.datastore.Suppress(testAllAccessContext, &start, &duration, "CVE-GHI")
	suite.NoError(err)
	// The Suppress method updates the cache automatically, no need to rebuild
	suite.datastore.EnrichNodeWithSuppressedCVEs(node)
	suite.verifySuppressionStateNode(node, []string{"CVE-ABC#", "CVE-DEF#", "CVE-GHI#"}, nil)

	// Clear image before unsupressing
	node = getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	// The Unsuppress method will handle the cache update internally
	err = suite.datastore.Unsuppress(testAllAccessContext, "CVE-GHI")
	suite.NoError(err)
	suite.datastore.EnrichNodeWithSuppressedCVEs(node)
	suite.verifySuppressionStateNode(node, []string{"CVE-ABC#", "CVE-DEF#"}, []string{"CVE-GHI#"})
}

func (suite *NodeCVEDataStoreSuite) TestUpsertMany() {
	// Test access denied
	err := suite.datastore.UpsertMany(sac.WithNoAccess(context.Background()), []*storage.NodeCVE{})
	suite.Require().Error(err)

	// Test successful upsert
	testCVE := &storage.NodeCVE{
		Id: "cve-1",
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-2021-0001",
		},
	}
	err = suite.datastore.UpsertMany(testAllAccessContext, []*storage.NodeCVE{testCVE})
	suite.Require().NoError(err)

	// Verify the CVE was actually stored
	stored, found, err := suite.datastore.Get(testAllAccessContext, "cve-1")
	suite.Require().NoError(err)
	suite.Require().True(found)
	suite.Equal("cve-1", stored.GetId())
	suite.Equal("CVE-2021-0001", stored.GetCveBaseInfo().GetCve())
}

func (suite *NodeCVEDataStoreSuite) TestPruneNodeCVEs() {
	// Test access denied
	err := suite.datastore.PruneNodeCVEs(sac.WithNoAccess(context.Background()), []string{})
	suite.Require().Error(err)

	// Set up test data first
	testCVEs := []*storage.NodeCVE{
		{
			Id: "cve-prune-1",
			CveBaseInfo: &storage.CVEInfo{
				Cve: "CVE-2021-0001",
			},
		},
		{
			Id: "cve-prune-2",
			CveBaseInfo: &storage.CVEInfo{
				Cve: "CVE-2021-0002",
			},
		},
	}
	err = suite.datastore.UpsertMany(testAllAccessContext, testCVEs)
	suite.Require().NoError(err)

	// Verify they exist
	for _, cve := range testCVEs {
		exists, err := suite.datastore.Exists(testAllAccessContext, cve.GetId())
		suite.Require().NoError(err)
		suite.True(exists)
	}

	// Test successful pruning
	err = suite.datastore.PruneNodeCVEs(testAllAccessContext, []string{"cve-prune-1", "cve-prune-2"})
	suite.Require().NoError(err)

	// Verify they were pruned
	for _, cve := range testCVEs {
		exists, err := suite.datastore.Exists(testAllAccessContext, cve.GetId())
		suite.Require().NoError(err)
		suite.False(exists)
	}
}

func TestGetSuppressionCacheEntry(t *testing.T) {
	startTime := time.Now().UTC()
	duration := 10 * time.Minute
	activation := startTime.Truncate(time.Nanosecond)
	expiration := startTime.Add(duration)

	protoStart, err := protocompat.ConvertTimeToTimestampOrError(startTime)
	assert.NoError(t, err)
	protoExpiration, err := protocompat.ConvertTimeToTimestampOrError(expiration)
	assert.NoError(t, err)

	cve1 := &storage.NodeCVE{}
	expectedEntry1 := common.SuppressionCacheEntry{}
	entry1 := getSuppressionCacheEntry(cve1)
	assert.Equal(t, expectedEntry1, entry1)

	cve2 := &storage.NodeCVE{
		SnoozeStart: protoStart,
	}
	expectedEntry2 := common.SuppressionCacheEntry{
		SuppressActivation: &activation,
	}
	entry2 := getSuppressionCacheEntry(cve2)
	assert.Equal(t, expectedEntry2, entry2)

	cve3 := &storage.NodeCVE{
		SnoozeExpiry: protoExpiration,
	}
	expectedEntry3 := common.SuppressionCacheEntry{
		SuppressExpiry: &expiration,
	}
	entry3 := getSuppressionCacheEntry(cve3)
	assert.Equal(t, expectedEntry3, entry3)

	cve4 := &storage.NodeCVE{
		SnoozeStart:  protoStart,
		SnoozeExpiry: protoExpiration,
	}
	expectedEntry4 := common.SuppressionCacheEntry{
		SuppressActivation: &activation,
		SuppressExpiry:     &expiration,
	}
	entry4 := getSuppressionCacheEntry(cve4)
	assert.Equal(t, expectedEntry4, entry4)
}

func TestGetSuppressExpiry(t *testing.T) {
	startTime := time.Now().UTC()
	duration := 10 * time.Minute

	expiry1, err := getSuppressExpiry(nil, nil)
	assert.ErrorIs(t, err, errNilSuppressionStart)
	assert.Nil(t, expiry1)

	expiry2, err := getSuppressExpiry(nil, &duration)
	assert.ErrorIs(t, err, errNilSuppressionStart)
	assert.Nil(t, expiry2)

	expiry3, err := getSuppressExpiry(&startTime, nil)
	assert.ErrorIs(t, err, errNilSuppressionDuration)
	assert.Nil(t, expiry3)

	expiry4, err := getSuppressExpiry(&startTime, &duration)
	assert.NoError(t, err)
	truncatedStart := startTime.Truncate(time.Second)
	truncatedDuration := duration.Truncate(time.Second)
	expectedExpiry4 := truncatedStart.Add(truncatedDuration)
	assert.Equal(t, &expectedExpiry4, expiry4)
}
