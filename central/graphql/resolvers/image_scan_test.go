package resolvers

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	imageCVEsDSMocks "github.com/stackrox/rox/central/cve/image/datastore/mocks"
	imageDSMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageComponentsDSMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	imageComponentsDSV2Mocks "github.com/stackrox/rox/central/imagecomponent/v2/datastore/mocks"
	imagesComponentViewMocks "github.com/stackrox/rox/central/views/imagecomponentflat/mocks"
	imagesView "github.com/stackrox/rox/central/views/images"
	imagesViewMocks "github.com/stackrox/rox/central/views/images/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	imageWithScanQuery = `
		query getImages($query: String, $pagination: Pagination) {
			images(query: $query, pagination: $pagination) { 
				id
				scan {
					imageComponents {
						name
						imageVulnerabilities {
							cve
						}
					}
				}
			}}`

	imageWithoutScanQuery = `
		query getImages($query: String, $pagination: Pagination) {
			images(query: $query, pagination: $pagination) { 
				id
				imageComponents {
					name
					imageVulnerabilities {
						cve
					}
				}
			}}`
)

func TestImageScanResolver(t *testing.T) {
	suite.Run(t, new(ImageScanResolverTestSuite))
}

type ImageScanResolverTestSuite struct {
	suite.Suite

	ctx      context.Context
	mockCtrl *gomock.Controller

	imageDataStore            *imageDSMocks.MockDataStore
	imageView                 *imagesViewMocks.MockImageView
	imageComponentDataStore   *imageComponentsDSMocks.MockDataStore
	imageComponentDataStoreV2 *imageComponentsDSV2Mocks.MockDataStore
	imageCVEDataStore         *imageCVEsDSMocks.MockDataStore
	imageComponentFlatView    *imagesComponentViewMocks.MockComponentFlatView

	resolver *Resolver
	schema   *graphql.Schema
}

func (s *ImageScanResolverTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithImagePerm(s.T(), s.mockCtrl)

	s.imageDataStore = imageDSMocks.NewMockDataStore(s.mockCtrl)
	s.imageView = imagesViewMocks.NewMockImageView(s.mockCtrl)
	s.imageComponentFlatView = imagesComponentViewMocks.NewMockComponentFlatView(s.mockCtrl)

	s.imageComponentDataStore = imageComponentsDSMocks.NewMockDataStore(s.mockCtrl)
	s.imageComponentDataStoreV2 = imageComponentsDSV2Mocks.NewMockDataStore(s.mockCtrl)
	s.imageCVEDataStore = imageCVEsDSMocks.NewMockDataStore(s.mockCtrl)

	s.resolver, s.schema = SetupTestResolver(s.T(), s.imageDataStore, s.imageView, s.imageComponentDataStore, s.imageCVEDataStore, s.imageComponentDataStoreV2, s.imageComponentFlatView)
}

func (s *ImageScanResolverTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ImageScanResolverTestSuite) TearDownSuite() {
}

func (s *ImageScanResolverTestSuite) TestGetImagesWithScan() {
	// Verify that full image is fetched.
	img := fixtures.GetImageWithUniqueComponents(5)
	imageCore := imagesViewMocks.NewMockImageCore(s.mockCtrl)
	imageCore.EXPECT().GetImageID().Return(img.GetId())
	s.imageView.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return([]imagesView.ImageCore{imageCore}, nil)
	cloned := img.CloneVT()
	cloned.GetScan().SetComponents(nil)
	s.imageDataStore.EXPECT().GetManyImageMetadata(gomock.Any(), gomock.Any()).
		Return([]*storage.Image{cloned}, nil)
	s.imageDataStore.EXPECT().GetImagesBatch(gomock.Any(), gomock.Any()).
		Return([]*storage.Image{img}, nil)
	response := s.schema.Exec(s.ctx, imageWithScanQuery, "getImages", nil)
	s.Len(response.Errors, 0)
}

func (s *ImageScanResolverTestSuite) TestGetImagesWithoutScan() {
	// Verify that full image is not fetched but rather image component and vuln stores are queried.
	img := fixtures.GetImageWithUniqueComponents(5)
	imageCore := imagesViewMocks.NewMockImageCore(s.mockCtrl)
	imageCore.EXPECT().GetImageID().Return(img.GetId())
	s.imageView.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return([]imagesView.ImageCore{imageCore}, nil)

	cloned := img.CloneVT()
	cloned.GetScan().SetComponents(nil)
	s.imageDataStore.EXPECT().GetManyImageMetadata(gomock.Any(), gomock.Any()).
		Return([]*storage.Image{cloned}, nil)
	if features.FlattenCVEData.Enabled() {
		s.imageComponentFlatView.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil)
		s.imageComponentDataStoreV2.EXPECT().SearchRawImageComponents(gomock.Any(), gomock.Any()).
			Return(nil, nil)
	} else {
		s.imageComponentDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).
			Return(nil, nil)
	}
	response := s.schema.Exec(s.ctx, imageWithoutScanQuery, "getImages", nil)
	s.Len(response.Errors, 0)
}
