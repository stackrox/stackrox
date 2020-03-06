package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	searchMock "github.com/stackrox/rox/central/image/datastore/internal/search/mocks"
	storeMock "github.com/stackrox/rox/central/image/datastore/internal/store/mocks"
	indexMock "github.com/stackrox/rox/central/image/index/mocks"
	componentDatastoreMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestImageDataStore(t *testing.T) {
	if features.Dackbox.Enabled() {
		return
	}
	suite.Run(t, new(ImageDataStoreTestSuite))
}

type ImageDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context

	mockIndexer  *indexMock.MockIndexer
	mockSearcher *searchMock.MockSearcher
	mockStore    *storeMock.MockStore

	datastore      DataStore
	mockComponents *componentDatastoreMocks.MockDataStore
	mockRisks      *riskDatastoreMocks.MockDataStore

	imageRanker *ranking.Ranker

	mockCtrl *gomock.Controller
}

func (suite *ImageDataStoreTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Image, resources.Risk)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image, resources.Risk)))

	suite.mockSearcher = searchMock.NewMockSearcher(suite.mockCtrl)
	suite.mockStore = storeMock.NewMockStore(suite.mockCtrl)
	suite.mockStore.EXPECT().GetKeysToIndex().Return(nil, nil)

	suite.mockIndexer = indexMock.NewMockIndexer(suite.mockCtrl)
	suite.mockIndexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	suite.mockComponents = componentDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockRisks = riskDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.imageRanker = ranking.NewRanker()

	var err error
	suite.datastore, err = newDatastoreImpl(suite.mockStore, suite.mockIndexer, suite.mockSearcher, suite.mockComponents, suite.mockRisks, suite.imageRanker)
	suite.Require().NoError(err)
}

func (suite *ImageDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

// Scenario: We have a new image with a sha and no scan or metadata. And no previously matched registry shas.
// Outcome: Image should be upserted and indexed unchanged.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithoutMetadata() {
	image := &storage.Image{
		Id: "sha1",
	}

	suite.mockStore.EXPECT().GetImage("sha1").Return((*storage.Image)(nil), false, nil)

	suite.mockStore.EXPECT().Upsert(image, nil).Return(nil)
	suite.mockIndexer.EXPECT().AddImage(image).Return(nil)
	suite.mockStore.EXPECT().AckKeysIndexed(image.GetId()).Return(nil)

	err := suite.datastore.UpsertImage(suite.hasWriteCtx, image)
	suite.NoError(err)
}

// Scenario: We have a new image with metadata, but its sha and the registry sha do not match.
// Outcome: The sha should be changed to the registry sha, and the mapping added to the store.
func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithMetadata() {
	newImage := &storage.Image{
		Id:       "sha1",
		Metadata: &storage.ImageMetadata{},
	}
	upsertedImage := &storage.Image{
		Id:       "sha1",
		Metadata: &storage.ImageMetadata{},
	}

	suite.mockStore.EXPECT().GetImage("sha1").Return((*storage.Image)(nil), false, nil)
	suite.mockStore.EXPECT().Upsert(upsertedImage, nil).Return(nil)
	suite.mockIndexer.EXPECT().AddImage(upsertedImage).Return(nil)
	suite.mockStore.EXPECT().AckKeysIndexed(upsertedImage.GetId()).Return(nil)

	err := suite.datastore.UpsertImage(suite.hasWriteCtx, newImage)
	suite.NoError(err)
}

func (suite *ImageDataStoreTestSuite) TestNewImageAddedWithScanStats() {
	newImage := &storage.Image{
		Id: "sha1",
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: "derp",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "v1.2",
							},
						},
						{
							Cve: "derp2",
						},
					},
				},
			},
		},
	}
	upsertedImage := &storage.Image{
		Id: "sha1",
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: "derp",
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "v1.2",
							},
						},
						{
							Cve: "derp2",
						},
					},
					SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{
						TopCvss: float32(0),
					},
				},
			},
		},
		SetComponents: &storage.Image_Components{
			Components: 1,
		},
		SetCves: &storage.Image_Cves{
			Cves: 2,
		},
		SetFixable: &storage.Image_FixableCves{
			FixableCves: 1,
		},
		SetTopCvss: &storage.Image_TopCvss{
			TopCvss: float32(0),
		},
	}

	suite.mockStore.EXPECT().GetImage("sha1").Return((*storage.Image)(nil), false, nil)
	suite.mockStore.EXPECT().Upsert(upsertedImage, nil).Return(nil)
	suite.mockIndexer.EXPECT().AddImage(upsertedImage).Return(nil)
	suite.mockStore.EXPECT().AckKeysIndexed(upsertedImage.GetId()).Return(nil)

	err := suite.datastore.UpsertImage(suite.hasWriteCtx, newImage)
	suite.NoError(err)
}

func (suite *ImageDataStoreTestSuite) TestDeleteImagesWithoutDackBox() {
	isolator := testutils.NewEnvIsolator(suite.T())
	isolator.Setenv(features.Dackbox.EnvVar(), "false")
	defer isolator.RestoreAll()

	suite.mockStore.EXPECT().Delete("id1").Return(nil)
	suite.mockIndexer.EXPECT().DeleteImage("id1").Return(nil)
	suite.mockStore.EXPECT().AckKeysIndexed("id1")

	suite.mockStore.EXPECT().Delete("id2").Return(nil)
	suite.mockIndexer.EXPECT().DeleteImage("id2").Return(nil)
	suite.mockStore.EXPECT().AckKeysIndexed("id2")

	suite.NoError(suite.datastore.DeleteImages(suite.hasWriteCtx, "id1", "id2"))
}

func TestImageReindexSuite(t *testing.T) {
	suite.Run(t, new(ImageReindexSuite))
}

type ImageReindexSuite struct {
	suite.Suite

	mockIndexer  *indexMock.MockIndexer
	mockSearcher *searchMock.MockSearcher
	mockStore    *storeMock.MockStore

	mockComponents *componentDatastoreMocks.MockDataStore
	mockRisks      *riskDatastoreMocks.MockDataStore

	imageRanker *ranking.Ranker

	ctx context.Context

	mockCtrl *gomock.Controller
}

func (suite *ImageReindexSuite) SetupTest() {
	suite.ctx = sac.WithAllAccess(context.Background())

	suite.mockCtrl = gomock.NewController(suite.T())

	suite.mockSearcher = searchMock.NewMockSearcher(suite.mockCtrl)
	suite.mockStore = storeMock.NewMockStore(suite.mockCtrl)
	suite.mockStore.EXPECT().GetKeysToIndex().Return(nil, nil)

	suite.mockIndexer = indexMock.NewMockIndexer(suite.mockCtrl)
	suite.mockIndexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	suite.mockComponents = componentDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockRisks = riskDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.imageRanker = ranking.NewRanker()
}

func (suite *ImageReindexSuite) TestReconciliationFullReindex() {
	suite.mockIndexer.EXPECT().NeedsInitialIndexing().Return(true, nil)

	img1 := fixtures.GetImage()
	img1.Id = "A"
	img2 := fixtures.GetImage()
	img2.Id = "B"

	suite.mockStore.EXPECT().GetImages().Return([]*storage.Image{img1, img2}, nil)
	suite.mockIndexer.EXPECT().AddImages([]*storage.Image{img1, img2}).Return(nil)

	suite.mockStore.EXPECT().GetKeysToIndex().Return([]string{"D", "E"}, nil)
	suite.mockStore.EXPECT().AckKeysIndexed([]string{"D", "E"}).Return(nil)

	suite.mockIndexer.EXPECT().MarkInitialIndexingComplete().Return(nil)

	_, err := newDatastoreImpl(suite.mockStore, suite.mockIndexer, suite.mockSearcher, suite.mockComponents, suite.mockRisks, suite.imageRanker)
	suite.Require().NoError(err)
}

func (suite *ImageReindexSuite) TestReconciliationPartialReindex() {
	suite.mockStore.EXPECT().GetKeysToIndex().Return([]string{"A", "B", "C"}, nil)
	suite.mockIndexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	img1 := fixtures.GetImage()
	img1.Id = "A"
	img2 := fixtures.GetImage()
	img2.Id = "B"
	img3 := fixtures.GetImage()
	img3.Id = "C"

	imageList := []*storage.Image{img1, img2, img3}

	suite.mockStore.EXPECT().GetImagesBatch([]string{"A", "B", "C"}).Return(imageList, nil, nil)
	suite.mockIndexer.EXPECT().AddImages(imageList).Return(nil)
	suite.mockStore.EXPECT().AckKeysIndexed([]string{"A", "B", "C"}).Return(nil)

	_, err := newDatastoreImpl(suite.mockStore, suite.mockIndexer, suite.mockSearcher, suite.mockComponents, suite.mockRisks, suite.imageRanker)
	suite.Require().NoError(err)

	// Make deploymentlist just A,B so C should be deleted
	imageList = imageList[:1]
	suite.mockStore.EXPECT().GetKeysToIndex().Return([]string{"A", "B", "C"}, nil)
	suite.mockIndexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	suite.mockStore.EXPECT().GetImagesBatch([]string{"A", "B", "C"}).Return(imageList, []int{2}, nil)
	suite.mockIndexer.EXPECT().AddImages(imageList).Return(nil)
	suite.mockIndexer.EXPECT().DeleteImages([]string{"C"}).Return(nil)
	suite.mockStore.EXPECT().AckKeysIndexed([]string{"A", "B", "C"}).Return(nil)

	_, err = newDatastoreImpl(suite.mockStore, suite.mockIndexer, suite.mockSearcher, suite.mockComponents, suite.mockRisks, suite.imageRanker)
	suite.Require().NoError(err)
}

func (suite *ImageReindexSuite) TestInitializeRanker() {
	ds, err := newDatastoreImpl(suite.mockStore, suite.mockIndexer, suite.mockSearcher, suite.mockComponents, suite.mockRisks, suite.imageRanker)
	suite.Require().NoError(err)

	images := []*storage.Image{
		{
			Id:        "1",
			RiskScore: float32(1.0),
		},
		{
			Id:        "2",
			RiskScore: float32(2.0),
		},
		{
			Id: "3",
		},
		{
			Id: "4",
		},
		{
			Id: "5",
		},
	}

	suite.mockSearcher.EXPECT().Search(gomock.Any(), search.EmptyQuery()).Return([]search.Result{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}, {ID: "5"}}, nil)
	suite.mockStore.EXPECT().GetImage(images[0].Id).Return(images[0], true, nil)
	suite.mockStore.EXPECT().GetImage(images[1].Id).Return(images[1], true, nil)
	suite.mockStore.EXPECT().GetImage(images[2].Id).Return(images[2], true, nil)
	suite.mockStore.EXPECT().GetImage(images[3].Id).Return(nil, false, nil)
	suite.mockStore.EXPECT().GetImage(images[4].Id).Return(nil, false, errors.New("fake error"))
	ds.initializeRankers()

	suite.Equal(int64(1), suite.imageRanker.GetRankForID("2"))
	suite.Equal(int64(2), suite.imageRanker.GetRankForID("1"))
	suite.Equal(int64(3), suite.imageRanker.GetRankForID("3"))
	suite.Equal(int64(3), suite.imageRanker.GetRankForID("4"))
	suite.Equal(int64(3), suite.imageRanker.GetRankForID("5"))
}
