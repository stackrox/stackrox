//go:build sql_integration

package datastore

import (
	"context"
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

func TestImageCVEV2DataStoreV1Filter(t *testing.T) {
	suite.Run(t, new(ImageCVEV2DataStoreV1FilterTestSuite))
}

type ImageCVEV2DataStoreV1FilterTestSuite struct {
	suite.Suite

	testDB   *pgtest.TestPostgres
	cveStore DataStore
	ctx      context.Context

	v2Image *storage.ImageV2
}

func (s *ImageCVEV2DataStoreV1FilterTestSuite) SetupSuite() {
	if !features.FlattenImageData.Enabled() {
		s.T().Skip("Skipping test because FlattenImageData feature flag is disabled")
	}
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	// Temporarily disable the flag to insert a V1 image.
	// The V1 store's CVE converter sets ImageId (not ImageIdV2) only when the flag is disabled.
	s.T().Setenv(features.FlattenImageData.EnvVar(), "false")
	v1Store := imageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	v1Image := fixtures.GetImageSherlockHolmes1()
	s.Require().NoError(v1Store.UpsertImage(s.ctx, v1Image))

	// Re-enable the flag for V2 image insertion and all subsequent queries.
	s.T().Setenv(features.FlattenImageData.EnvVar(), "true")
	v2Store := imageV2DS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.v2Image = imageUtils.ConvertToV2(fixtures.GetImageDoctorJekyll2())
	s.Require().NoError(v2Store.UpsertImage(s.ctx, s.v2Image))

	s.cveStore = GetTestPostgresDataStore(s.T(), s.testDB.DB)
}

func (s *ImageCVEV2DataStoreV1FilterTestSuite) expectedV2CVEs() set.StringSet {
	cves := set.NewStringSet()
	for _, comp := range s.v2Image.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			cves.Add(vuln.GetCve())
		}
	}
	return cves
}

func (s *ImageCVEV2DataStoreV1FilterTestSuite) TestCountExcludesV1() {
	expectedCVEs := s.expectedV2CVEs()

	count, err := s.cveStore.Count(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Assert().Equal(expectedCVEs.Cardinality(), count)
}

func (s *ImageCVEV2DataStoreV1FilterTestSuite) TestSearchExcludesV1() {
	expectedCVEs := s.expectedV2CVEs()

	results, err := s.cveStore.Search(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Assert().Equal(expectedCVEs.Cardinality(), len(results))
}

func (s *ImageCVEV2DataStoreV1FilterTestSuite) TestSearchRawImageCVEsExcludesV1() {
	expectedCVEs := s.expectedV2CVEs()

	cves, err := s.cveStore.SearchRawImageCVEs(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)

	returnedCVEs := set.NewStringSet()
	for _, cve := range cves {
		returnedCVEs.Add(cve.GetCveBaseInfo().GetCve())
	}
	s.Assert().Equal(expectedCVEs, returnedCVEs)
}

func (s *ImageCVEV2DataStoreV1FilterTestSuite) TestSearchImageCVEsExcludesV1() {
	expectedCVEs := s.expectedV2CVEs()

	results, err := s.cveStore.SearchImageCVEs(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Assert().Equal(expectedCVEs.Cardinality(), len(results))
}
