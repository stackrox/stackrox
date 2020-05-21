package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	indexMocks "github.com/stackrox/rox/central/cve/index/mocks"
	searchMocks "github.com/stackrox/rox/central/cve/search/mocks"
	storeMocks "github.com/stackrox/rox/central/cve/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	graphMocks "github.com/stackrox/rox/pkg/dackbox/graph/mocks"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

var (
	testSuppressionQuery = searchPkg.NewQueryBuilder().AddBools(searchPkg.CVESuppressed, true).ProtoQuery()

	testAllAccessContext = sac.WithAllAccess(context.Background())
)

func TestCVEDataStore(t *testing.T) {
	suite.Run(t, new(CVEDataStoreSuite))
}

type CVEDataStoreSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	indexer   *indexMocks.MockIndexer
	storage   *storeMocks.MockStore
	searcher  *searchMocks.MockSearcher
	provider  *graphMocks.MockProvider
	datastore *datastoreImpl
}

func (suite *CVEDataStoreSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.indexer = indexMocks.NewMockIndexer(suite.mockCtrl)
	suite.storage = storeMocks.NewMockStore(suite.mockCtrl)
	suite.searcher = searchMocks.NewMockSearcher(suite.mockCtrl)
	suite.provider = graphMocks.NewMockProvider(suite.mockCtrl)

	suite.searcher.EXPECT().SearchRawCVEs(getCVECtx, testSuppressionQuery).Return([]*storage.CVE{}, nil)

	ds, err := New(suite.provider, suite.storage, suite.indexer, suite.searcher)
	suite.Require().NoError(err)
	suite.datastore = ds.(*datastoreImpl)
}

func (suite *CVEDataStoreSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
}

func getImageWithCVEs(cves ...string) *storage.Image {
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(cves))
	for _, cve := range cves {
		vulns = append(vulns, &storage.EmbeddedVulnerability{
			Cve: cve,
		})
	}
	return &storage.Image{
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Vulns: vulns,
				},
			},
		},
	}
}

func (suite *CVEDataStoreSuite) verifySuppressionState(image *storage.Image, suppressedCVEs, unsuppressedCVEs []string) {
	cveMap := make(map[string]bool)
	for _, comp := range image.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			cveMap[vuln.Cve] = vuln.GetSuppressed()
		}
	}
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

func (suite *CVEDataStoreSuite) TestSuppressionCache() {
	// Add some results
	suite.searcher.EXPECT().SearchRawCVEs(getCVECtx, testSuppressionQuery).Return([]*storage.CVE{
		{
			Id:         "CVE-ABC",
			Suppressed: true,
		},
		{
			Id:         "CVE-DEF",
			Suppressed: true,
		},
	}, nil)
	suite.NoError(suite.datastore.buildSuppressedCache())
	expectedCache := map[string]suppressionCacheEntry{
		"CVE-ABC": {
			Suppressed: true,
		},
		"CVE-DEF": {
			Suppressed: true,
		},
	}
	suite.Equal(expectedCache, suite.datastore.cveSuppressionCache)

	// No apply these to the image
	img := getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionState(img, []string{"CVE-ABC", "CVE-DEF"}, []string{"CVE-GHI"})

	start := types.TimestampNow()
	duration := types.DurationProto(10 * time.Minute)

	expiry, err := getSuppressExpiry(start, duration)
	suite.NoError(err)

	suite.storage.EXPECT().GetBatch([]string{"CVE-GHI"}).Return([]*storage.CVE{{Id: "CVE-GHI"}}, nil, nil)
	storedCVE := &storage.CVE{
		Id:                 "CVE-GHI",
		Suppressed:         true,
		SuppressActivation: start,
		SuppressExpiry:     expiry,
	}
	suite.storage.EXPECT().Upsert(storedCVE).Return(nil)

	// Clear image before suppressing
	img = getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	err = suite.datastore.Suppress(testAllAccessContext, start, duration, "CVE-GHI")
	suite.NoError(err)
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionState(img, []string{"CVE-ABC", "CVE-DEF", "CVE-GHI"}, nil)

	// Clear image before unsupressing
	img = getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.storage.EXPECT().GetBatch([]string{"CVE-GHI"}).Return([]*storage.CVE{storedCVE}, nil, nil)
	suite.storage.EXPECT().Upsert(&storage.CVE{Id: "CVE-GHI"}).Return(nil)
	err = suite.datastore.Unsuppress(testAllAccessContext, "CVE-GHI")
	suite.NoError(err)
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionState(img, []string{"CVE-ABC", "CVE-DEF"}, []string{"CVE-GHI"})
}
