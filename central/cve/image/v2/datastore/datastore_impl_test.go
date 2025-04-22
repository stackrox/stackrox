package datastore

import (
	"testing"

	searchMocks "github.com/stackrox/rox/central/cve/image/datastore/search/mocks"
	storeMocks "github.com/stackrox/rox/central/cve/image/datastore/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
	ds := New(suite.storage, suite.searcher)
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
