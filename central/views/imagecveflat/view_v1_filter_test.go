//go:build sql_integration

package imagecveflat

import (
	"context"
	"os"
	"testing"

	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

func TestImageCVEFlatViewV1Filter(t *testing.T) {
	suite.Run(t, new(ImageCVEFlatViewV1FilterTestSuite))
}

type ImageCVEFlatViewV1FilterTestSuite struct {
	suite.Suite

	testDB  *pgtest.TestPostgres
	cveView CveFlatView
	ctx     context.Context

	v2Image *storage.ImageV2
}

func (s *ImageCVEFlatViewV1FilterTestSuite) SetupSuite() {
	if !features.FlattenImageData.Enabled() {
		s.T().Skip("Skipping test because FlattenImageData feature flag is disabled")
	}
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	// Temporarily disable the flag to insert a V1 image.
	// The V1 store's CVE converter sets ImageId (not ImageIdV2) only when the flag is disabled.
	s.Require().NoError(os.Setenv(features.FlattenImageData.EnvVar(), "false"))
	v1Store := imageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	v1Image := fixtures.GetImageSherlockHolmes1()
	s.Require().NoError(v1Store.UpsertImage(s.ctx, v1Image))

	// Re-enable the flag for V2 image insertion and all subsequent queries.
	s.Require().NoError(os.Setenv(features.FlattenImageData.EnvVar(), "true"))
	v2Store := imageV2DS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.v2Image = imageUtils.ConvertToV2(fixtures.GetImageDoctorJekyll2())
	s.Require().NoError(v2Store.UpsertImage(s.ctx, s.v2Image))

	s.cveView = NewCVEFlatView(s.testDB.DB)
}

func (s *ImageCVEFlatViewV1FilterTestSuite) expectedV2CVEs() set.StringSet {
	cves := set.NewStringSet()
	for _, comp := range s.v2Image.GetScan().GetComponents() {
		for _, vuln := range comp.GetVulns() {
			cves.Add(vuln.GetCve())
		}
	}
	return cves
}

func (s *ImageCVEFlatViewV1FilterTestSuite) TestCountExcludesV1() {
	expectedCVEs := s.expectedV2CVEs()

	count, err := s.cveView.Count(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Assert().Equal(expectedCVEs.Cardinality(), count)
}

func (s *ImageCVEFlatViewV1FilterTestSuite) TestGetExcludesV1() {
	expectedCVEs := s.expectedV2CVEs()

	q := search.EmptyQuery()
	q.Pagination = &v1.QueryPagination{Limit: 100}

	results, err := s.cveView.Get(s.ctx, q, views.ReadOptions{})
	s.Require().NoError(err)
	s.Assert().Len(results, expectedCVEs.Cardinality())

	returnedCVEs := set.NewStringSet()
	for _, cveFlat := range results {
		returnedCVEs.Add(cveFlat.GetCVE())
	}
	s.Assert().Equal(expectedCVEs, returnedCVEs)
}
