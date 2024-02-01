//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	cveDS "github.com/stackrox/rox/central/cve/image/datastore"
	imageCVEDS "github.com/stackrox/rox/central/cve/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestFixableSearch(t *testing.T) {
	suite.Run(t, new(FixableSearchTestSuite))
}

type FixableSearchTestSuite struct {
	suite.Suite

	ctx    context.Context
	testDB *pgtest.TestPostgres

	imageDataStore DataStore
	cveDataStore   imageCVEDS.DataStore
}

func (s *FixableSearchTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())

	imageDataStore := GetTestPostgresDataStore(s.T(), s.testDB)
	s.imageDataStore = imageDataStore
	cveDataStore := cveDS.GetTestPostgresDataStore(s.T(), s.testDB)
	s.cveDataStore = cveDataStore

	s.ctx = sac.WithAllAccess(context.Background())
	for _, image := range fixableSearchTestImages() {
		s.Require().NoError(s.imageDataStore.UpsertImage(s.ctx, image))
	}
}

func (s *FixableSearchTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *FixableSearchTestSuite) TestImageSearch() {
	for _, tc := range []struct {
		desc     string
		q        *v1.Query
		expected []string
		skip     bool
	}{
		{
			desc: "Search all images with at least some fixable vulnerabilities",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).ProtoQuery(),
			expected: []string{"image-1", "image-2", "image-4"},
		},
		{
			desc: "Search all images where 'cve-1' is fixable",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.CVE, "cve-1").
				ProtoQuery(),
			expected: []string{"image-1", "image-4"},
			skip:     true, // TODO: Enable once https://issues.redhat.com/browse/ROX-17252 is addressed.
		},
		{
			desc: "Search all images with at least some non-fixable vulnerabilities",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, false).ProtoQuery(),
			expected: []string{"image-2", "image-3", "image-4"},
		},
		{
			desc: "Search all images where 'cve-2' is not fixable vulnerabilities",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, false).
				AddExactMatches(search.CVE, "cve-2").ProtoQuery(),
			expected: []string{"image-3", "image-4"},
			skip:     true, // TODO: Enable once https://issues.redhat.com/browse/ROX-17252 is addressed.
		},
		{
			desc: "Search all images with at least some fixable vulnerabilities that are deferred",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
			expected: []string{"image-2", "image-4"},
		},
		{
			desc: "Search all images where 'cve-1' is fixable and deferred",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.CVE, "cve-1").
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
			expected: []string{"image-4"},
		},
		{
			desc: "Search all images where 'cve-1' is fixable in component 'comp-1'",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.Component, "comp-1").
				AddExactMatches(search.CVE, "cve-1").ProtoQuery(),
			expected: []string{"image-1", "image-4"},
			skip:     true, // TODO: Enable once https://issues.redhat.com/browse/ROX-17252 is addressed.
		},
		{
			desc: "Search all images where 'cve-1' is deferred and fixable in component 'comp-1'",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.Component, "comp-1").
				AddExactMatches(search.CVE, "cve-1").
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
			expected: []string{"image-4"},
		},
		{
			desc: "Search all images where 'cve-1' is fixable and observed",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.CVE, "cve-1").
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).ProtoQuery(),
			expected: []string{"image-1"},
			skip:     true, // TODO: Enable once https://issues.redhat.com/browse/ROX-17252 is addressed.
		},
		{
			desc: "Search all images",
			q: search.DisjunctionQuery(search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.CVE, "cve-1").
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
				search.NewQueryBuilder().
					AddBools(search.Fixable, true).
					AddExactMatches(search.CVE, "cve-2").
					AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).ProtoQuery(),
			),
			expected: []string{"image-4", "image-1"},
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			if tc.skip {
				t.SkipNow()
			}
			results, err := s.imageDataStore.Search(s.ctx, tc.q)
			s.NoError(err)
			actual := search.ResultsToIDs(results)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func (s *FixableSearchTestSuite) TestCVESearch() {
	for _, tc := range []struct {
		desc     string
		q        *v1.Query
		expected []string
		skip     bool
	}{
		{
			desc: "Search all fixable CVEs",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).ProtoQuery(),
			expected: []string{"cve-1#", "cve-2#"},
		},
		{
			desc: "Search all CVEs in 'image-2' that are fixable",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.ImageSHA, "image-2").
				ProtoQuery(),
			expected: []string{"cve-2#"},
			skip:     true, // TODO: Enable once https://issues.redhat.com/browse/ROX-17252 is addressed.
		},
		{
			desc: "Search CVE 'cve-1' which is not fixable in 'image-2' but fixable elsewhere",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.CVE, "cve-1").
				AddExactMatches(search.ImageSHA, "image-2").ProtoQuery(),
			expected: []string{},
			skip:     true, // TODO: Enable once https://issues.redhat.com/browse/ROX-17252 is addressed.
		},
		{
			desc: "Search all CVEs in 'image-4' that are fixable",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.ImageSHA, "image-4").ProtoQuery(),
			expected: []string{"cve-1#", "cve-2#"},
		},
		{
			desc: "Search all CVEs that are not fixable",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, false).ProtoQuery(),
			expected: []string{"cve-1#", "cve-2#"},
		},
		{
			desc: "Search all CVEs that are fixable and deferred",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
			expected: []string{"cve-1#", "cve-2#"},
		},
		{
			desc: "Search CVE 'cve-1' is fixable and deferred",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.CVE, "cve-1").
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
			expected: []string{"cve-1#"},
		},
		{
			desc: "Search CVE 'cve-1' is fixable and deferred",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.CVE, "cve-1").
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
			expected: []string{"cve-1#"},
		},
		{
			desc: "Search all CVEs that are fixable and deferred in 'image-1'",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.ImageSHA, "image-1").
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
			expected: []string{},
		},
		{
			desc: "Search all CVEs fixable in component 'comp-1'",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.Component, "comp-1").ProtoQuery(),
			expected: []string{"cve-1#", "cve-2#"},
		},
		{
			desc: "Search CVE 'cve-1' not fixable in component 'comp-2' but fixable elsewhere",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.Component, "comp-2").
				AddExactMatches(search.CVE, "cve-1").ProtoQuery(),
			expected: []string{},
		},
		{
			desc: "Search all CVEs that are fixable and observed in 'image-1'",
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				AddExactMatches(search.ImageSHA, "image-1").
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).ProtoQuery(),
			expected: []string{"cve-1#", "cve-2#"},
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			if tc.skip {
				t.SkipNow()
			}
			results, err := s.cveDataStore.Search(s.ctx, tc.q)
			s.NoError(err)
			actual := search.ResultsToIDs(results)
			assert.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func fixableSearchTestImages() []*storage.Image {
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
								Cve: "cve-1",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "ver-2",
								},
								State: storage.VulnerabilityState_OBSERVED,
							},
							{
								Cve: "cve-2",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "ver-3",
								},
								State: storage.VulnerabilityState_OBSERVED,
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
						Version: "ver-3",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:   "cve-1",
								State: storage.VulnerabilityState_OBSERVED,
							},
							{
								Cve: "cve-2",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "ver-3",
								},
								State: storage.VulnerabilityState_DEFERRED,
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
								Cve:   "cve-1",
								State: storage.VulnerabilityState_OBSERVED,
							},
							{
								Cve:   "cve-2",
								State: storage.VulnerabilityState_OBSERVED,
							},
						},
					},
				},
			},
		},
		{
			Id: "image-4",
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp-1",
						Version: "ver-1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "cve-1",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "ver-2",
								},
								State: storage.VulnerabilityState_DEFERRED,
							},
							{
								Cve: "cve-2",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "ver-3",
								},
								State: storage.VulnerabilityState_DEFERRED,
							},
						},
					},
					{
						Name:    "comp-2",
						Version: "ver-1",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:   "cve-1",
								State: storage.VulnerabilityState_DEFERRED,
							},
							{
								Cve:   "cve-2",
								State: storage.VulnerabilityState_DEFERRED,
							},
						},
					},
				},
			},
		},
	}
}
