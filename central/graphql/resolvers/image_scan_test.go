package resolvers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/graph-gophers/graphql-go"
	imageCVEsDSMocks "github.com/stackrox/rox/central/cve/image/datastore/mocks"
	imageDSMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageComponentsDSMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
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

	ctx         context.Context
	envIsolator *envisolator.EnvIsolator
	mockCtrl    *gomock.Controller

	imageDataStore          *imageDSMocks.MockDataStore
	imageComponentDataStore *imageComponentsDSMocks.MockDataStore
	imageCVEDataStore       *imageCVEsDSMocks.MockDataStore

	resolver *Resolver
	schema   *graphql.Schema
}

func (s *ImageScanResolverTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}
}

func (s *ImageScanResolverTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = contextWithImagePerm(s.T(), s.mockCtrl)

	s.imageDataStore = imageDSMocks.NewMockDataStore(s.mockCtrl)
	s.imageComponentDataStore = imageComponentsDSMocks.NewMockDataStore(s.mockCtrl)
	s.imageCVEDataStore = imageCVEsDSMocks.NewMockDataStore(s.mockCtrl)

	s.resolver, s.schema = setupResolver(s.T(), s.imageDataStore, s.imageComponentDataStore, s.imageCVEDataStore, nil)
}

func (s *ImageScanResolverTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ImageScanResolverTestSuite) TearDownSuite() {
	s.envIsolator.RestoreAll()
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
