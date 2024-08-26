package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cve/common"
	searchMocks "github.com/stackrox/rox/central/cve/node/datastore/search/mocks"
	storeMocks "github.com/stackrox/rox/central/cve/node/datastore/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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

	mockCtrl *gomock.Controller

	storage   *storeMocks.MockStore
	searcher  *searchMocks.MockSearcher
	datastore *datastoreImpl
}

func (suite *NodeCVEDataStoreSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.storage = storeMocks.NewMockStore(suite.mockCtrl)
	suite.searcher = searchMocks.NewMockSearcher(suite.mockCtrl)

	suite.searcher.EXPECT().SearchRawCVEs(accessAllCtx, testSuppressionQuery).Return([]*storage.NodeCVE{}, nil)

	ds, err := New(suite.storage, suite.searcher, concurrency.NewKeyFence())
	suite.Require().NoError(err)
	suite.datastore = ds.(*datastoreImpl)
}

func (suite *NodeCVEDataStoreSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
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
	// Add some results
	suite.searcher.EXPECT().SearchRawCVEs(accessAllCtx, testSuppressionQuery).Return([]*storage.NodeCVE{
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
	}, nil)
	suite.NoError(suite.datastore.buildSuppressedCache())
	expectedCache := common.CVESuppressionCache{
		"CVE-ABC": {},
		"CVE-DEF": {},
	}
	suite.Equal(expectedCache, suite.datastore.cveSuppressionCache)

	// No apply these to the image
	node := getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.datastore.EnrichNodeWithSuppressedCVEs(node)
	suite.verifySuppressionStateNode(node, []string{"CVE-ABC#", "CVE-DEF#"}, []string{"CVE-GHI#"})

	start := time.Now()
	duration := 10 * time.Minute

	expiry, err := getSuppressExpiry(&start, &duration)
	suite.NoError(err)

	suite.searcher.EXPECT().SearchRawCVEs(testAllAccessContext, gomock.Any()).Return(
		[]*storage.NodeCVE{
			{
				Id: "CVE-GHI",
				CveBaseInfo: &storage.CVEInfo{
					Cve: "CVE-GHI",
				},
			},
		}, nil)
	storedCVE := &storage.NodeCVE{
		Id: "CVE-GHI",
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-GHI",
		},
		Snoozed:      true,
		SnoozeStart:  protocompat.ConvertTimeToTimestampOrNil(&start),
		SnoozeExpiry: protocompat.ConvertTimeToTimestampOrNil(expiry),
	}
	suite.storage.EXPECT().UpsertMany(testAllAccessContext, []*storage.NodeCVE{storedCVE}).Return(nil)

	// Clear image before suppressing
	node = getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	err = suite.datastore.Suppress(testAllAccessContext, &start, &duration, "CVE-GHI")
	suite.NoError(err)
	suite.datastore.EnrichNodeWithSuppressedCVEs(node)
	suite.verifySuppressionStateNode(node, []string{"CVE-ABC#", "CVE-DEF#", "CVE-GHI#"}, nil)

	// Clear image before unsupressing
	node = getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.searcher.EXPECT().SearchRawCVEs(testAllAccessContext, gomock.Any()).Return([]*storage.NodeCVE{storedCVE}, nil)
	suite.storage.EXPECT().UpsertMany(testAllAccessContext, []*storage.NodeCVE{
		{Id: "CVE-GHI", CveBaseInfo: &storage.CVEInfo{Cve: "CVE-GHI"}},
	}).Return(nil)
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
	suite.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.NodeCVE{{Id: "cve-1"}}).Times(1).Return(nil)
	err = suite.datastore.UpsertMany(testAllAccessContext, []*storage.NodeCVE{{Id: "cve-1"}})
	suite.Require().NoError(err)

	// Test error from storage
	suite.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.NodeCVE{{Id: "cve-2"}}).Times(1).Return(errors.New("upsert failed"))
	err = suite.datastore.UpsertMany(testAllAccessContext, []*storage.NodeCVE{{Id: "cve-2"}})
	suite.Require().Error(err)
}

func (suite *NodeCVEDataStoreSuite) TestPruneNodeCVEs() {
	// Test access denied
	err := suite.datastore.PruneNodeCVEs(sac.WithNoAccess(context.Background()), []string{})
	suite.Require().Error(err)

	// Test successful pruning
	suite.storage.EXPECT().PruneMany(gomock.Any(), []string{"cve-1", "cve-2"}).Times(1).Return(nil)
	err = suite.datastore.PruneNodeCVEs(testAllAccessContext, []string{"cve-1", "cve-2"})
	suite.Require().NoError(err)

	// Test error from storage
	suite.storage.EXPECT().PruneMany(gomock.Any(), []string{"cve-3", "cve-4"}).Times(1).Return(errors.New("prune failed"))
	err = suite.datastore.PruneNodeCVEs(testAllAccessContext, []string{"cve-3", "cve-4"})
	suite.Require().Error(err)
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
