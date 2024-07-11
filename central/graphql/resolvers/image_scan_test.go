package resolvers

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	imageCVEsDSMocks "github.com/stackrox/rox/central/cve/image/datastore/mocks"
	imageDSMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageComponentsDSMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
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

	imageDataStore          *imageDSMocks.MockDataStore
	imageComponentDataStore *imageComponentsDSMocks.MockDataStore
	imageCVEDataStore       *imageCVEsDSMocks.MockDataStore

	resolver *Resolver
	schema   *graphql.Schema
}

func (s *ImageScanResolverTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithImagePerm(s.T(), s.mockCtrl)

	s.imageDataStore = imageDSMocks.NewMockDataStore(s.mockCtrl)
	s.imageComponentDataStore = imageComponentsDSMocks.NewMockDataStore(s.mockCtrl)
	s.imageCVEDataStore = imageCVEsDSMocks.NewMockDataStore(s.mockCtrl)

	s.resolver, s.schema = SetupTestResolver(s.T(), s.imageDataStore, s.imageComponentDataStore, s.imageCVEDataStore)
}

func (s *ImageScanResolverTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ImageScanResolverTestSuite) TearDownSuite() {
}

func (s *ImageScanResolverTestSuite) TestGetImagesWithScan() {
	// Verify that full image is fetched.
	img := fixtures.GetImageWithUniqueComponents(5)
	s.imageDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).
		Return([]search.Result{{
			ID: img.GetId(),
		}}, nil)
	cloned := img.Clone()
	cloned.Scan.Components = nil
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
	s.imageDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).
		Return([]search.Result{{
			ID: img.GetId(),
		}}, nil)

	cloned := img.Clone()
	cloned.Scan.Components = nil
	s.imageDataStore.EXPECT().GetManyImageMetadata(gomock.Any(), gomock.Any()).
		Return([]*storage.Image{cloned}, nil)
	s.imageComponentDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).
		Return(nil, nil)
	response := s.schema.Exec(s.ctx, imageWithoutScanQuery, "getImages", nil)
	s.Len(response.Errors, 0)
}
