package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/cve/common"
	searchMocks "github.com/stackrox/rox/central/cve/image/datastore/search/mocks"
	storeMocks "github.com/stackrox/rox/central/cve/image/datastore/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	testSuppressionQuery = searchPkg.NewQueryBuilder().AddBools(searchPkg.CVESuppressed, true).ProtoQuery()

	testAllAccessContext = sac.WithAllAccess(context.Background())
)

func TestImageCVEDataStore(t *testing.T) {
	suite.Run(t, new(ImageCVEDataStoreSuite))
}

type ImageCVEDataStoreSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	storage   *storeMocks.MockStore
	searcher  *searchMocks.MockSearcher
	datastore *datastoreImpl
}

func (suite *ImageCVEDataStoreSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.storage = storeMocks.NewMockStore(suite.mockCtrl)
	suite.searcher = searchMocks.NewMockSearcher(suite.mockCtrl)

	suite.searcher.EXPECT().SearchRawImageCVEs(accessAllCtx, testSuppressionQuery).Return([]*storage.ImageCVE{}, nil)

	ds := New(suite.storage, suite.searcher, concurrency.NewKeyFence())
	suite.datastore = ds.(*datastoreImpl)
}

func (suite *ImageCVEDataStoreSuite) TearDownSuite() {
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

func (suite *ImageCVEDataStoreSuite) verifySuppressionStateImage(image *storage.Image, suppressedCVEs, unsuppressedCVEs []string) {
	cveMap := make(map[string]bool)
	for _, comp := range image.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			cveMap[vuln.Cve] = vuln.GetSuppressed()
		}
	}
	suite.verifySuppressionState(cveMap, suppressedCVEs, unsuppressedCVEs)
}

func (suite *ImageCVEDataStoreSuite) verifySuppressionState(cveMap map[string]bool, suppressedCVEs, unsuppressedCVEs []string) {
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

func (suite *ImageCVEDataStoreSuite) TestSuppressionCacheImages() {
	// Add some results
	suite.searcher.EXPECT().SearchRawImageCVEs(accessAllCtx, testSuppressionQuery).Return([]*storage.ImageCVE{
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
	suite.datastore.buildSuppressedCache()
	expectedCache := common.CVESuppressionCache{
		"CVE-ABC": {},
		"CVE-DEF": {},
	}
	suite.Equal(expectedCache, suite.datastore.cveSuppressionCache)

	// No apply these to the image
	img := getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionStateImage(img, []string{"CVE-ABC", "CVE-DEF"}, []string{"CVE-GHI"})

	start := types.TimestampNow()
	duration := types.DurationProto(10 * time.Minute)

	expiry, err := getSuppressExpiry(start, duration)
	suite.NoError(err)

	suite.searcher.EXPECT().SearchRawImageCVEs(testAllAccessContext, gomock.Any()).Return([]*storage.ImageCVE{{CveBaseInfo: &storage.CVEInfo{Cve: "CVE-GHI"}}}, nil)
	storedCVE := &storage.ImageCVE{
		CveBaseInfo: &storage.CVEInfo{
			Cve: "CVE-GHI",
		},
		Snoozed:      true,
		SnoozeStart:  start,
		SnoozeExpiry: expiry,
	}
	suite.storage.EXPECT().UpsertMany(testAllAccessContext, []*storage.ImageCVE{storedCVE}).Return(nil)

	// Clear image before suppressing
	img = getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	err = suite.datastore.Suppress(testAllAccessContext, start, duration, "CVE-GHI")
	suite.NoError(err)
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionStateImage(img, []string{"CVE-ABC", "CVE-DEF", "CVE-GHI"}, nil)

	// Clear image before unsupressing
	img = getImageWithCVEs("CVE-ABC", "CVE-DEF", "CVE-GHI")
	suite.searcher.EXPECT().SearchRawImageCVEs(testAllAccessContext, gomock.Any()).Return([]*storage.ImageCVE{storedCVE}, nil)
	suite.storage.EXPECT().UpsertMany(testAllAccessContext, []*storage.ImageCVE{{CveBaseInfo: &storage.CVEInfo{Cve: "CVE-GHI"}}}).Return(nil)
	err = suite.datastore.Unsuppress(testAllAccessContext, "CVE-GHI")
	suite.NoError(err)
	suite.datastore.EnrichImageWithSuppressedCVEs(img)
	suite.verifySuppressionStateImage(img, []string{"CVE-ABC", "CVE-DEF"}, []string{"CVE-GHI"})
}
