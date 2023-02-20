//go:build sql_integration

package imagecve

import (
	"context"
	"testing"

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestGetImageCVECore(t *testing.T) {
	t.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		t.Skipf("Requires %s=true. Skipping the test", env.PostgresDatastoreEnabled.EnvVar())
		t.SkipNow()
	}

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	defer testDB.Teardown(t)

	store, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	images, err := imageSamples.GetTestImages(t)
	require.NoError(t, err)
	for _, image := range images {
		require.NoError(t, store.UpsertImage(ctx, image))
	}

	cveView := NewCVEView(testDB.DB)

	for _, tc := range []struct {
		desc        string
		q           *v1.Query
		expectedErr string
		expected    []*imageCVECore
	}{
		{
			desc:     "search all",
			q:        search.NewQueryBuilder().ProtoQuery(),
			expected: compileExpected(images, matchAllFilter()),
		},
		{
			desc: "search one cve",
			q:    search.NewQueryBuilder().AddExactMatches(search.CVE, "CVE-2022-1552").ProtoQuery(),
			expected: compileExpected(images,
				matchAllFilter().withVulnFiler(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.GetCve() == "CVE-2022-1552"
				}),
			),
		},
		{
			desc: "search one image",
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:latest").ProtoQuery(),
			expected: compileExpected(images,
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
			expected: compileExpected(images,
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
			expected: compileExpected(images,
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
			expected: compileExpected(images,
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
			expected: compileExpected(images,
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
			expected: compileExpected(images,
				matchAllFilter().withImageFiler(func(image *storage.Image) bool {
					return image.GetScan().GetOperatingSystem() == "debian:8"
				}),
			),
		},
		{
			desc:     "no match",
			q:        search.NewQueryBuilder().AddExactMatches(search.OperatingSystem, "").ProtoQuery(),
			expected: compileExpected(images, matchNoneFilter()),
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
	} {
		t.Run(tc.desc, func(t *testing.T) {
			actual, err := cveView.Get(ctx, tc.q)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, len(tc.expected), len(actual))
			assert.ElementsMatch(t, tc.expected, actual)
		})
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

				val := cveMap[vuln.GetCve()]
				if val == nil {
					val = &imageCVECore{
						CVE: vuln.GetCve(),
					}
					cveMap[val.CVE] = val
				}
				val.TopCVSS = mathutil.MaxFloat32(val.TopCVSS, vuln.GetCvss())
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
