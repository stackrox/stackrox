//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	imageCVEDS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TestOriginFlatSearch is the flag-off counterpart of TestOriginSearch: it
// exercises the origin column write path in the legacy image datastore's
// copyFromImageComponentV2Cves (central/image/datastore/store/v2/postgres).
// TODO(ROX-30117): Remove this test when FlattenImageData feature flag is removed.
func TestOriginFlatSearch(t *testing.T) {
	if features.FlattenImageData.Enabled() {
		t.Skip("Skipping test - FlattenImageData is enabled. Equivalent tests exist in imageV2 datastore.")
	}
	suite.Run(t, new(OriginFlatSearchTestSuite))
}

type OriginFlatSearchTestSuite struct {
	suite.Suite

	ctx    context.Context
	testDB *pgtest.TestPostgres

	imageDataStore DataStore
	cveDataStore   imageCVEDS.DataStore
}

func (s *OriginFlatSearchTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())

	s.imageDataStore = GetTestPostgresDataStore(s.T(), s.testDB)
	s.cveDataStore = imageCVEDS.GetTestPostgresDataStore(s.T(), s.testDB)

	s.ctx = sac.WithAllAccess(context.Background())
	for _, image := range originSearchTestImages() {
		s.Require().NoError(s.imageDataStore.UpsertImage(s.ctx, image))
	}
}

func (s *OriginFlatSearchTestSuite) TestImageSearch() {
	for _, tc := range []struct {
		desc     string
		q        *v1.Query
		expected []string
	}{
		{
			desc: "Images with a Red Hat origin CVE",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_RED_HAT.String()).ProtoQuery(),
			expected: []string{"image-1"},
		},
		{
			desc: "Images with an OSV origin CVE",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_OSV.String()).ProtoQuery(),
			expected: []string{"image-1", "image-2"},
		},
		{
			desc: "Images with an Ubuntu origin CVE",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_UBUNTU.String()).ProtoQuery(),
			expected: []string{"image-3"},
		},
		{
			desc: "Red Hat origin CVE scoped to component 'comp-1'",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEOrigin, storage.VulnOrigin_VULN_ORIGIN_RED_HAT.String()).
				AddExactMatches(search.Component, "comp-1").ProtoQuery(),
			expected: []string{"image-1"},
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

func (s *OriginFlatSearchTestSuite) TestCVESearch() {
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

// originSearchTestImages seeds three images whose CVEs carry distinct origins:
//   - image-1: cve-1 (Red Hat), cve-2 (OSV)
//   - image-2: cve-2 (OSV)
//   - image-3: cve-3 (Ubuntu)
func originSearchTestImages() []*storage.Image {
	return []*storage.Image{
		{
			Id: "image-1",
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
			Id: "image-2",
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
			Id: "image-3",
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
