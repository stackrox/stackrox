package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/cve/common"
	searchMocks "github.com/stackrox/rox/central/cve/node/datastore/search/mocks"
	storeMocks "github.com/stackrox/rox/central/cve/node/datastore/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
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

	start := types.TimestampNow()
	duration := types.DurationProto(10 * time.Minute)

	expiry, err := getSuppressExpiry(start, duration)
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
		SnoozeStart:  start,
		SnoozeExpiry: expiry,
	}
	suite.storage.EXPECT().UpsertMany(testAllAccessContext, []*storage.NodeCVE{storedCVE}).Return(nil)

	// Clear image before suppressing
	node = getNodeWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	err = suite.datastore.Suppress(testAllAccessContext, start, duration, "CVE-GHI")
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
