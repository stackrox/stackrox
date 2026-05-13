//go:build sql_integration

package datastore

import (
	"context"
	"os"
	"testing"

	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
)

func TestImageComponentV2DataStoreV1Filter(t *testing.T) {
	suite.Run(t, new(ImageComponentV2DataStoreV1FilterTestSuite))
}

type ImageComponentV2DataStoreV1FilterTestSuite struct {
	suite.Suite

	testDB         *pgtest.TestPostgres
	componentStore DataStore
	ctx            context.Context

	v2Image *storage.ImageV2
}

func (s *ImageComponentV2DataStoreV1FilterTestSuite) SetupSuite() {
	if !features.FlattenImageData.Enabled() {
		s.T().Skip("Skipping test because FlattenImageData feature flag is disabled")
	}
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	// The CVE converter sets ImageId vs ImageIdV2 based on the FlattenImageData flag, so temporarily disable it to insert a V1 image.
	s.Require().NoError(os.Setenv(features.FlattenImageData.EnvVar(), "false"))
	v1Store := imageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(v1Store.UpsertImage(s.ctx, fixtures.GetImageSherlockHolmes1()))

	s.Require().NoError(os.Setenv(features.FlattenImageData.EnvVar(), "true"))
	v2Store := imageV2DS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.v2Image = imageUtils.ConvertToV2(fixtures.GetImageDoctorJekyll2())
	s.Require().NoError(v2Store.UpsertImage(s.ctx, s.v2Image))

	s.componentStore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
}

func (s *ImageComponentV2DataStoreV1FilterTestSuite) expectedV2Components() set.StringSet {
	components := set.NewStringSet()
	for _, comp := range s.v2Image.GetScan().GetComponents() {
		components.Add(comp.GetName())
	}
	return components
}

func (s *ImageComponentV2DataStoreV1FilterTestSuite) TestCountExcludesV1() {
	expectedComponents := s.expectedV2Components()

	count, err := s.componentStore.Count(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Assert().Equal(expectedComponents.Cardinality(), count)
}

func (s *ImageComponentV2DataStoreV1FilterTestSuite) TestSearchExcludesV1() {
	expectedComponents := s.expectedV2Components()

	results, err := s.componentStore.Search(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Assert().Equal(expectedComponents.Cardinality(), len(results))
}

func (s *ImageComponentV2DataStoreV1FilterTestSuite) TestSearchRawImageComponentsExcludesV1() {
	expectedComponents := s.expectedV2Components()

	components, err := s.componentStore.SearchRawImageComponents(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)

	returnedComponents := set.NewStringSet()
	for _, comp := range components {
		returnedComponents.Add(comp.GetName())
	}
	s.Assert().Equal(expectedComponents, returnedComponents)
}

func (s *ImageComponentV2DataStoreV1FilterTestSuite) TestSearchImageComponentsExcludesV1() {
	expectedComponents := s.expectedV2Components()

	results, err := s.componentStore.SearchImageComponents(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Assert().Equal(expectedComponents.Cardinality(), len(results))
}
