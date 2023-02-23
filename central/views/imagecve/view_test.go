//go:build sql_integration

package imagecve

import (
	"context"
	"sort"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	imageSamples "github.com/stackrox/rox/pkg/fixtures/image"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
)

type testCase struct {
	desc        string
	q           *v1.Query
	expectedErr string
	expected    []*imageCVECore
}

type filterImpl struct {
	matchImage func(image *storage.Image) bool
	matchVuln  func(vuln *storage.EmbeddedVulnerability) bool
}

func matchAllFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ *storage.Image) bool {
			return true
		},
		matchVuln: func(_ *storage.EmbeddedVulnerability) bool {
			return true
		},
	}
}

func matchNoneFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ *storage.Image) bool {
			return false
		},
		matchVuln: func(_ *storage.EmbeddedVulnerability) bool {
			return false
		},
	}
}

func (f *filterImpl) withImageFiler(fn func(image *storage.Image) bool) *filterImpl {
	f.matchImage = fn
	return f
}

func (f *filterImpl) withVulnFiler(fn func(vuln *storage.EmbeddedVulnerability) bool) *filterImpl {
	f.matchVuln = fn
	return f
}

func TestImageCVEView(t *testing.T) {
	suite.Run(t, new(ImageCVEViewTestSuite))
}

type ImageCVEViewTestSuite struct {
	suite.Suite

	ctx     context.Context
	testDB  *pgtest.TestPostgres
	cveView CveView

	testImages []*storage.Image
}

func (s *ImageCVEViewTestSuite) SetupSuite() {
	s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skipf("Requires %s=true. Skipping the test", env.PostgresDatastoreEnabled.EnvVar())
		s.T().SkipNow()
	}

	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	store, err := datastore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	images, err := imageSamples.GetTestImages(s.T())
	s.Require().NoError(err)
	for _, image := range images {
		s.Require().NoError(store.UpsertImage(s.ctx, image))
	}

	// Ensure that the image is stored and constructed as expected.
	for idx, image := range images {
		actual, found, err := store.GetImage(s.ctx, image.GetId())
		s.Require().NoError(err)
		s.Require().True(found)

		cloned := actual.Clone()
		// Adjust dynamic fields and ensure images in ACS are as expected.
		s.Require().EqualValues(standardizeImages(image), standardizeImages(cloned))

		// Now that we confirmed that images match, use stored image to establish the expected test results.
		// This makes dynamic fields matching (e.g. created at) straightforward.
		images[idx] = actual
	}

	s.testImages = images
	s.cveView = NewCVEView(s.testDB.DB)
}

func (s *ImageCVEViewTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ImageCVEViewTestSuite) TestGetImageCVECore() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.Get(s.ctx, tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			s.NoError(err)
			s.Equal(len(tc.expected), len(actual))
			s.ElementsMatch(tc.expected, actual)
		})
	}
}

func (s *ImageCVEViewTestSuite) TestCountImageCVECore() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.Count(s.ctx, tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			s.NoError(err)
			s.Equal(len(tc.expected), actual)
		})
	}
}

func (s *ImageCVEViewTestSuite) testCases() []testCase {
	return []testCase{
		{
			desc:     "search all",
			q:        search.NewQueryBuilder().ProtoQuery(),
			expected: compileExpected(s.testImages, matchAllFilter()),
		},
		{
			desc: "search one cve",
			q:    search.NewQueryBuilder().AddExactMatches(search.CVE, "CVE-2022-1552").ProtoQuery(),
			expected: compileExpected(s.testImages,
				matchAllFilter().withVulnFiler(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.GetCve() == "CVE-2022-1552"
				}),
			),
		},
		{
			desc: "search one image",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:latest").ProtoQuery(),
			expected: compileExpected(s.testImages,
				matchAllFilter().withImageFiler(func(image *storage.Image) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:latest"
				}),
			),
		},
		{
			desc: "search one cve + one image",
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2022-1552").
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:debian").
				ProtoQuery(),
			expected: compileExpected(s.testImages,
				matchAllFilter().
					withImageFiler(func(image *storage.Image) bool {
						return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian"
					}).
					withVulnFiler(func(vuln *storage.EmbeddedVulnerability) bool {
						return vuln.GetCve() == "CVE-2022-1552"
					}),
			),
		},
		{
			desc: "search critical severity",
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).
				ProtoQuery(),
			expected: compileExpected(s.testImages,
				matchAllFilter().
					withVulnFiler(func(vuln *storage.EmbeddedVulnerability) bool {
						return vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
					}),
			),
		},
		{
			desc: "search multiple severities",
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity,
					storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
				).
				ProtoQuery(),
			expected: compileExpected(s.testImages,
				matchAllFilter().
					withVulnFiler(func(vuln *storage.EmbeddedVulnerability) bool {
						return vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY ||
							vuln.GetSeverity() == storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
					}),
			),
		},
		{
			desc: "search critical severity + one image",
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:debian").
				ProtoQuery(),
			expected: compileExpected(s.testImages,
				matchAllFilter().
					withImageFiler(func(image *storage.Image) bool {
						return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian"
					}).
					withVulnFiler(func(vuln *storage.EmbeddedVulnerability) bool {
						return vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
					}),
			),
		},
		{
			desc: "search one operating system",
			q:    search.NewQueryBuilder().AddExactMatches(search.OperatingSystem, "debian:8").ProtoQuery(),
			expected: compileExpected(s.testImages,
				matchAllFilter().withImageFiler(func(image *storage.Image) bool {
					return image.GetScan().GetOperatingSystem() == "debian:8"
				}),
			),
		},
		{
			desc:     "no match",
			q:        search.NewQueryBuilder().AddExactMatches(search.OperatingSystem, "").ProtoQuery(),
			expected: compileExpected(s.testImages, matchNoneFilter()),
		},
		{
			desc: "with select",
			q: search.NewQueryBuilder().
				AddSelectFields(&v1.QueryField{Field: search.CVE.String()}).
				AddExactMatches(search.OperatingSystem, "").ProtoQuery(),
			expectedErr: "Unexpected select clause in query",
		},
		{
			desc: "with group by",
			q: search.NewQueryBuilder().
				AddExactMatches(search.OperatingSystem, "").
				AddGroupBy(search.CVE).ProtoQuery(),
			expectedErr: "Unexpected group by clause in query",
		},
	}
}

func compileExpected(images []*storage.Image, filter *filterImpl) []*imageCVECore {
	cveMap := make(map[string]*imageCVECore)

	for _, image := range images {
		if !filter.matchImage(image) {
			continue
		}

		var seenForImage set.Set[string]
		for _, component := range image.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				if !filter.matchVuln(vuln) {
					continue
				}

				vulnTime, _ := types.TimestampFromProto(vuln.GetFirstSystemOccurrence())
				val := cveMap[vuln.GetCve()]
				if val == nil {
					val = &imageCVECore{
						CVE:                     vuln.GetCve(),
						TopCVSS:                 vuln.GetCvss(),
						FirstDiscoveredInSystem: vulnTime,
					}
					cveMap[val.CVE] = val
				}

				val.TopCVSS = mathutil.MaxFloat32(val.GetTopCVSS(), vuln.GetCvss())
				if val.GetFirstDiscoveredInSystem().After(vulnTime) {
					val.FirstDiscoveredInSystem = vulnTime
				}

				if seenForImage.Add(val.CVE) {
					val.AffectedImages++
				}
			}
		}
	}

	ret := make([]*imageCVECore, 0, len(cveMap))
	for _, entry := range cveMap {
		ret = append(ret, entry)
	}
	return ret
}

func standardizeImages(images ...*storage.Image) {
	for _, image := range images {
		if image.GetMetadata().GetV1() != nil && len(image.GetMetadata().GetV1().GetLabels()) == 0 {
			image.Metadata.V1.Labels = nil
		}

		components := image.GetScan().GetComponents()
		for _, component := range components {
			component.Priority = 0
			if len(component.GetVulns()) == 0 {
				component.Vulns = nil
			}

			vulns := component.GetVulns()
			for _, vuln := range vulns {
				vuln.FirstImageOccurrence = nil
				vuln.FirstSystemOccurrence = nil
			}

			sort.SliceStable(vulns, func(i, j int) bool {
				return vulns[i].Cve < vulns[j].Cve
			})
		}

		sort.SliceStable(components, func(i, j int) bool {
			if components[i].Name == components[j].Name {
				return components[i].Version < components[j].Version
			}
			return components[i].Name < components[j].Name
		})
	}
}
