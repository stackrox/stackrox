//go:build sql_integration

package datastoretest

import (
	"context"
	"testing"

	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	"github.com/stackrox/rox/central/imagev2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TestOriginSearch verifies that the CVE origin enum is written to the
// dedicated image_cves_v2.origin column on image upsert and is filterable.
// The column (not the serialized image blob) backs the "CVE Origin" search
// field, so this is the only coverage that actually exercises the origin
// write path in copyFromImageComponentV2Cves - a plain store round-trip via
// Get() reads the blob and would pass even if the column were never written.
func TestOriginSearch(t *testing.T) {
	if !features.FlattenImageData.Enabled() {
		t.Skip("Image flattened data model is not enabled")
	}
	suite.Run(t, new(OriginSearchTestSuite))
}

type OriginSearchTestSuite struct {
	suite.Suite

	ctx    context.Context
	testDB *pgtest.TestPostgres

	imageDataStore datastore.DataStore
	cveDataStore   imageCVEDS.DataStore
}

var (
	originImage1 = uuid.NewV5FromNonUUIDs("registry.test.io/image-1:latest", "sha256:image-1").String()
	originImage2 = uuid.NewV5FromNonUUIDs("registry.test.io/image-2:latest", "sha256:image-2").String()
	originImage3 = uuid.NewV5FromNonUUIDs("registry.test.io/image-3:latest", "sha256:image-3").String()
)

func (s *OriginSearchTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())

	s.imageDataStore = datastore.GetTestPostgresDataStore(s.T(), s.testDB)
	s.cveDataStore = imageCVEDS.GetTestPostgresDataStore(s.T(), s.testDB)

	s.ctx = sac.WithAllAccess(context.Background())
	for _, image := range originSearchTestImagesV2() {
		s.Require().NoError(s.imageDataStore.UpsertImage(s.ctx, image))
	}
}

func (s *OriginSearchTestSuite) TestImageSearch() {
	for _, tc := range []struct {
		desc     string
		q        *v1.Query
		expected []string
	}{
		{
			desc: "Images with a Red Hat origin CVE",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_RED_HAT.String()).ProtoQuery(),
			expected: []string{originImage1},
		},
		{
			desc: "Images with an OSV origin CVE",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_OSV.String()).ProtoQuery(),
			expected: []string{originImage1, originImage2},
		},
		{
			desc: "Images with an Ubuntu origin CVE",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_UBUNTU.String()).ProtoQuery(),
			expected: []string{originImage3},
		},
		{
			desc: "Red Hat origin CVE scoped to component 'comp-1'",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_RED_HAT.String()).
				AddExactMatches(search.Component, "comp-1").ProtoQuery(),
			expected: []string{originImage1},
		},
		{
			desc: "No images with an Amazon origin CVE",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_AMAZON.String()).ProtoQuery(),
			expected: []string{},
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			results, err := s.imageDataStore.Search(s.ctx, tc.q)
			s.NoError(err)
			actual := search.ResultsToIDs(results)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func (s *OriginSearchTestSuite) TestCVESearch() {
	for _, tc := range []struct {
		desc     string
		q        *v1.Query
		expected []string
	}{
		{
			desc: "Red Hat origin CVEs",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_RED_HAT.String()).ProtoQuery(),
			expected: []string{"cve-1"},
		},
		{
			desc: "OSV origin CVEs",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_OSV.String()).ProtoQuery(),
			expected: []string{"cve-2"},
		},
		{
			desc: "Ubuntu origin CVEs",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_UBUNTU.String()).ProtoQuery(),
			expected: []string{"cve-3"},
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			results, err := s.cveDataStore.Search(s.ctx, tc.q)
			s.NoError(err)
			actual := splitFlattenedIDs(search.ResultsToIDs(results))
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}

// originSearchTestImagesV2 seeds three images whose CVEs carry distinct
// origins so filtering by "CVE Origin" is deterministic:
//   - image-1: cve-1 (Red Hat), cve-2 (OSV)
//   - image-2: cve-2 (OSV)
//   - image-3: cve-3 (Ubuntu)
func originSearchTestImagesV2() []*storage.ImageV2 {
	return []*storage.ImageV2{
		{
			Id:     originImage1,
			Digest: "sha256:image-1",
			Name: &storage.ImageName{
				Registry: "registry.test.io",
				Remote:   "image-1",
				Tag:      "latest",
				FullName: "registry.test.io/image-1:latest",
			},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp-1",
						Version: "ver-1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:    "cve-1",
								Origin: storage.VulnOrigin_VULN_ORIGIN_RED_HAT,
								State:  storage.VulnerabilityState_OBSERVED,
							},
							{
								Cve:    "cve-2",
								Origin: storage.VulnOrigin_VULN_ORIGIN_OSV,
								State:  storage.VulnerabilityState_OBSERVED,
							},
						},
					},
				},
			},
		},
		{
			Id:     originImage2,
			Digest: "sha256:image-2",
			Name: &storage.ImageName{
				Registry: "registry.test.io",
				Remote:   "image-2",
				Tag:      "latest",
				FullName: "registry.test.io/image-2:latest",
			},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp-1",
						Version: "ver-1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:    "cve-2",
								Origin: storage.VulnOrigin_VULN_ORIGIN_OSV,
								State:  storage.VulnerabilityState_OBSERVED,
							},
						},
					},
				},
			},
		},
		{
			Id:     originImage3,
			Digest: "sha256:image-3",
			Name: &storage.ImageName{
				Registry: "registry.test.io",
				Remote:   "image-3",
				Tag:      "latest",
				FullName: "registry.test.io/image-3:latest",
			},
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp-2",
						Version: "ver-1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:    "cve-3",
								Origin: storage.VulnOrigin_VULN_ORIGIN_UBUNTU,
								State:  storage.VulnerabilityState_OBSERVED,
							},
						},
					},
				},
			},
		},
	}
}
