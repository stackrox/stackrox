//go:build sql_integration

package images

import (
	"context"
	"sort"
	"testing"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type lessFunc func(records []*imageResponse) func(i, j int) bool

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

func (f *filterImpl) withImageFilter(fn func(image *storage.Image) bool) *filterImpl {
	f.matchImage = fn
	return f
}

func (f *filterImpl) withVulnFilter(fn func(vuln *storage.EmbeddedVulnerability) bool) *filterImpl {
	f.matchVuln = fn
	return f
}

func TestImageView(t *testing.T) {
	suite.Run(t, new(ImageViewTestSuite))
}

type ImageViewTestSuite struct {
	suite.Suite

	testDB          *pgtest.TestPostgres
	imagesView      ImageView
	testImagesMap   map[string]*storage.Image
	scopeToImageIDs map[string]set.StringSet
}

func (s *ImageViewTestSuite) SetupSuite() {
	ctx := sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	// Initialize the datastores
	imageStore := imageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	deploymentStore, err := deploymentDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	// Upsert test images
	images := testImages()
	for _, image := range images {
		s.Require().NoError(imageStore.UpsertImage(ctx, image))
	}

	s.testImagesMap = make(map[string]*storage.Image)
	for _, image := range images {
		actual, found, err := imageStore.GetImage(ctx, image.GetId())
		s.Require().NoError(err)
		s.Require().True(found)
		s.testImagesMap[actual.GetId()] = actual
	}

	s.imagesView = NewImageView(s.testDB.DB)

	deployments := []*storage.Deployment{
		fixtures.GetDeploymentWithImage(testconsts.Cluster1, testconsts.NamespaceA, s.testImagesMap["sha1"]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster1, testconsts.NamespaceA, s.testImagesMap["sha1"]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, s.testImagesMap["sha2"]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, s.testImagesMap["sha3"]),
	}
	for _, d := range deployments {
		s.Require().NoError(deploymentStore.UpsertDeployment(ctx, d))
	}

	s.scopeToImageIDs = map[string]set.StringSet{
		testutils.UnrestrictedReadWriteCtx:       set.NewStringSet("sha1", "sha2", "sha3", "sha4"),
		testutils.Cluster1ReadWriteCtx:           set.NewStringSet("sha1"),
		testutils.Cluster1NamespaceAReadWriteCtx: set.NewStringSet("sha1"),
		testutils.Cluster2ReadWriteCtx:           set.NewStringSet("sha2", "sha3"),
		testutils.Cluster2NamespaceBReadWriteCtx: set.NewStringSet("sha2", "sha3"),
		testutils.Cluster3ReadWriteCtx:           set.NewStringSet(),
		testutils.Cluster3NamespaceCReadWriteCtx: set.NewStringSet(),
	}
}

func (s *ImageViewTestSuite) TestGetImagesCore() {
	contextMap := testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
	for _, tc := range []struct {
		ctx                     context.Context
		desc                    string
		query                   *v1.Query
		ignoreOrder             bool
		isError                 bool
		hasVulnFilter           bool
		hasSortBySeverityCounts bool
		matchFilter             *filterImpl
		less                    lessFunc
	}{
		{
			ctx:         contextMap[testutils.UnrestrictedReadWriteCtx],
			desc:        "search all",
			query:       search.EmptyQuery(),
			ignoreOrder: true,
			matchFilter: matchAllFilter(),
		},
		{
			ctx:     contextMap[testutils.UnrestrictedReadWriteCtx],
			desc:    "invalid query",
			query:   search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.ImageName)).ProtoQuery(),
			isError: true,
		},
		{
			ctx:  contextMap[testutils.UnrestrictedReadWriteCtx],
			desc: "order by risk score",
			query: search.NewQueryBuilder().
				WithPagination(
					search.NewPagination().AddSortOption(search.NewSortOption(search.ImageRiskScore).Reversed(true)),
				).
				ProtoQuery(),
			matchFilter: matchAllFilter(),
			less: func(records []*imageResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					scorei := s.testImagesMap[records[i].ImageID].RiskScore
					scorej := s.testImagesMap[records[j].ImageID].RiskScore
					if scorei == scorej {
						return records[i].ImageID < records[j].ImageID
					}
					return scorei > scorej
				}
			},
		},
		{
			ctx:  contextMap[testutils.UnrestrictedReadWriteCtx],
			desc: "filtered query with order by risk score",
			query: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				WithPagination(
					search.NewPagination().AddSortOption(search.NewSortOption(search.ImageRiskScore).Reversed(true)),
				).
				ProtoQuery(),
			hasVulnFilter: true,
			matchFilter: matchAllFilter().withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
				return vuln.GetFixedBy() != ""
			}),
			less: func(records []*imageResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					scorei := s.testImagesMap[records[i].ImageID].RiskScore
					scorej := s.testImagesMap[records[j].ImageID].RiskScore
					if scorei == scorej {
						return records[i].ImageID < records[j].ImageID
					}
					return scorei > scorej
				}
			},
		},
		{
			ctx:  contextMap[testutils.UnrestrictedReadWriteCtx],
			desc: "order by number of critical CVEs",
			query: search.NewQueryBuilder().
				WithPagination(
					search.NewPagination().AddSortOption(search.NewSortOption(search.CriticalSeverityCount).Reversed(true)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter:             matchAllFilter(),
			less: func(records []*imageResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if records[i].CriticalSeverityCount == records[j].CriticalSeverityCount {
						return records[i].ImageID < records[j].ImageID
					}
					return records[i].CriticalSeverityCount > records[j].CriticalSeverityCount
				}
			},
		},
		{
			ctx:  contextMap[testutils.UnrestrictedReadWriteCtx],
			desc: "multi sort by severity counts",
			query: search.NewQueryBuilder().
				WithPagination(
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.CriticalSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.ImportantSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.ModerateSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter:             matchAllFilter(),
			less: func(records []*imageResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if !(records[i].CriticalSeverityCount == records[j].CriticalSeverityCount) {
						return records[i].CriticalSeverityCount > records[j].CriticalSeverityCount
					}
					if !(records[i].ImportantSeverityCount == records[j].ImportantSeverityCount) {
						return records[i].ImportantSeverityCount > records[j].ImportantSeverityCount
					}
					if !(records[i].ModerateSeverityCount == records[j].ModerateSeverityCount) {
						return records[i].ModerateSeverityCount > records[j].ModerateSeverityCount
					}
					if !(records[i].LowSeverityCount == records[j].LowSeverityCount) {
						return records[i].LowSeverityCount > records[j].LowSeverityCount
					}
					return records[i].ImageID < records[j].ImageID
				}
			},
		},
		{
			ctx:  contextMap[testutils.UnrestrictedReadWriteCtx],
			desc: "filtered query with multi sort by severity counts",
			query: search.NewQueryBuilder().
				AddExactMatches(search.Severity,
					storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
					storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
				).
				WithPagination(
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.CriticalSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.ImportantSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.ModerateSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)),
				).
				ProtoQuery(),
			hasVulnFilter:           true,
			hasSortBySeverityCounts: true,
			matchFilter: matchAllFilter().withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
				return vuln.Severity == storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY ||
					vuln.Severity == storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
			}),
			less: func(records []*imageResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if !(records[i].ModerateSeverityCount == records[j].ModerateSeverityCount) {
						return records[i].ModerateSeverityCount > records[j].ModerateSeverityCount
					}
					if !(records[i].LowSeverityCount == records[j].LowSeverityCount) {
						return records[i].LowSeverityCount > records[j].LowSeverityCount
					}
					return records[i].ImageID < records[j].ImageID
				}
			},
		},
		{
			ctx:           contextMap[testutils.UnrestrictedReadWriteCtx],
			desc:          "filtered query",
			query:         search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery(),
			ignoreOrder:   true,
			hasVulnFilter: true,
			matchFilter: matchAllFilter().withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
				return vuln.GetFixedBy() != ""
			}),
		},
		{
			ctx:         contextMap[testutils.Cluster1ReadWriteCtx],
			desc:        "cluster-1 scoped context",
			query:       search.EmptyQuery(),
			ignoreOrder: true,
			matchFilter: matchAllFilter().withImageFilter(func(image *storage.Image) bool {
				return s.scopeToImageIDs[testutils.Cluster1ReadWriteCtx].Contains(image.GetId())
			}),
		},
		{
			ctx:         contextMap[testutils.Cluster1NamespaceAReadWriteCtx],
			desc:        "cluster-1 namespace-A scoped context",
			query:       search.EmptyQuery(),
			ignoreOrder: true,
			matchFilter: matchAllFilter().withImageFilter(func(image *storage.Image) bool {
				return s.scopeToImageIDs[testutils.Cluster1NamespaceAReadWriteCtx].Contains(image.GetId())
			}),
		},
		{
			ctx:  contextMap[testutils.Cluster2ReadWriteCtx],
			desc: "cluster-2 scoped context and query multi-sort by severity counts",
			query: search.NewQueryBuilder().
				WithPagination(
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.ModerateSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter: matchAllFilter().withImageFilter(func(image *storage.Image) bool {
				return s.scopeToImageIDs[testutils.Cluster2ReadWriteCtx].Contains(image.GetId())
			}),
			less: func(records []*imageResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if !(records[i].ModerateSeverityCount == records[j].ModerateSeverityCount) {
						return records[i].ModerateSeverityCount > records[j].ModerateSeverityCount
					}
					if !(records[i].LowSeverityCount == records[j].LowSeverityCount) {
						return records[i].LowSeverityCount > records[j].LowSeverityCount
					}
					return records[i].ImageID < records[j].ImageID
				}
			},
		},
		{
			ctx:  contextMap[testutils.Cluster2NamespaceBReadWriteCtx],
			desc: "cluster-2 namespace-B scoped context and query multi-sort by severity counts",
			query: search.NewQueryBuilder().
				WithPagination(
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.ModerateSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter: matchAllFilter().withImageFilter(func(image *storage.Image) bool {
				return s.scopeToImageIDs[testutils.Cluster2NamespaceBReadWriteCtx].Contains(image.GetId())
			}),
			less: func(records []*imageResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if !(records[i].ModerateSeverityCount == records[j].ModerateSeverityCount) {
						return records[i].ModerateSeverityCount > records[j].ModerateSeverityCount
					}
					if !(records[i].LowSeverityCount == records[j].LowSeverityCount) {
						return records[i].LowSeverityCount > records[j].LowSeverityCount
					}
					return records[i].ImageID < records[j].ImageID
				}
			},
		},
		{
			ctx:         contextMap[testutils.Cluster3ReadWriteCtx],
			desc:        "cluster-3 scoped context",
			query:       search.EmptyQuery(),
			ignoreOrder: true,
			matchFilter: matchNoneFilter(),
		},
		{
			ctx:         contextMap[testutils.Cluster3NamespaceCReadWriteCtx],
			desc:        "cluster-3 namespace-C scoped context",
			query:       search.EmptyQuery(),
			ignoreOrder: true,
			matchFilter: matchNoneFilter(),
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			imageCores, err := s.imagesView.Get(tc.ctx, tc.query)
			if tc.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			expectedCores := s.compileExpected(tc.matchFilter, tc.less, tc.hasVulnFilter, tc.hasSortBySeverityCounts)
			if tc.ignoreOrder {
				assert.ElementsMatch(t, expectedCores, imageCores)
			} else {
				assert.Equal(t, expectedCores, imageCores)
			}
		})
	}
}

func (s *ImageViewTestSuite) compileExpected(filter *filterImpl, less lessFunc, hasVulnFilters, hasSortBySeverityCounts bool) []ImageCore {
	expected := make([]*imageResponse, 0)
	for _, image := range s.testImagesMap {
		if !filter.matchImage(image) {
			continue
		}

		val := &imageResponse{
			ImageID: image.GetId(),
		}
		if !hasVulnFilters && !hasSortBySeverityCounts {
			expected = append(expected, val)
			continue
		}

		matchedVulnCount := 0
		for _, component := range image.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				if !filter.matchVuln(vuln) {
					continue
				}
				matchedVulnCount += 1

				if hasSortBySeverityCounts {
					switch vuln.GetSeverity() {
					case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
						val.CriticalSeverityCount += 1
						if vuln.GetFixedBy() != "" {
							val.FixableCriticalSeverityCount += 1
						}
					case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
						val.ImportantSeverityCount += 1
						if vuln.GetFixedBy() != "" {
							val.FixableImportantSeverityCount += 1
						}
					case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
						val.ModerateSeverityCount += 1
						if vuln.GetFixedBy() != "" {
							val.FixableModerateSeverityCount += 1
						}
					case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
						val.LowSeverityCount += 1
						if vuln.GetFixedBy() != "" {
							val.FixableLowSeverityCount += 1
						}
					}
				}
			}
		}
		if matchedVulnCount > 0 {
			expected = append(expected, val)
		}
	}

	if less != nil {
		sort.SliceStable(expected, less(expected))
	}
	ret := make([]ImageCore, 0, len(expected))
	for _, r := range expected {
		ret = append(ret, r)
	}
	return ret
}

func testImages() []*storage.Image {
	img1 := &storage.Image{
		Id: "sha1",
		Name: &storage.ImageName{
			Registry: "reg1",
			Remote:   "img1",
			Tag:      "tag1",
			FullName: "reg1/img1:tag1",
		},
		SetCves: &storage.Image_Cves{
			Cves: 6,
		},
		RiskScore: 10,
		Scan: &storage.ImageScan{
			OperatingSystem: "os1",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "comp1",
					Version: "0.9",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:      "cve-1",
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-2",
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-3",
							Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-4",
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-5",
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-6",
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						},
					},
				},
			},
		},
	}

	img2 := &storage.Image{
		Id: "sha2",
		Name: &storage.ImageName{
			Registry: "reg1",
			Remote:   "img1",
			Tag:      "tag2",
			FullName: "reg1/img1:tag2",
		},
		SetCves: &storage.Image_Cves{
			Cves: 6,
		},
		RiskScore: 9,
		Scan: &storage.ImageScan{
			OperatingSystem: "os2",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "comp1",
					Version: "0.9",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:      "cve-1",
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-2",
							Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.1",
							},
						},
						{
							Cve:      "cve-3",
							Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-4",
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.1",
							},
						},
						{
							Cve:      "cve-5",
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-6",
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						},
					},
				},
			},
		},
	}

	img3 := &storage.Image{
		Id: "sha3",
		Name: &storage.ImageName{
			Registry: "reg3",
			Remote:   "img3",
			Tag:      "tag3",
			FullName: "reg3/img3:tag3",
		},
		SetCves: &storage.Image_Cves{
			Cves: 6,
		},
		RiskScore: 8,
		Scan: &storage.ImageScan{
			OperatingSystem: "os1",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "comp3",
					Version: "0.9",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:      "cve-7",
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.1",
							},
						},
						{
							Cve:      "cve-8",
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-9",
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.1",
							},
						},
						{
							Cve:      "cve-10",
							Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-11",
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-12",
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
					},
				},
			},
		},
	}

	img4 := &storage.Image{
		Id: "sha4",
		Name: &storage.ImageName{
			Registry: "reg4",
			Remote:   "img4",
			Tag:      "tag4",
			FullName: "reg4/img4:tag4",
		},
		SetCves: &storage.Image_Cves{
			Cves: 6,
		},
		RiskScore: 7,
		Scan: &storage.ImageScan{
			OperatingSystem: "os1",
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "comp4",
					Version: "0.9",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:      "cve-13",
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-14",
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-15",
							Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-16",
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-17",
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						},
						{
							Cve:      "cve-18",
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
						},
					},
				},
			},
		},
	}
	return []*storage.Image{img1, img2, img3, img4}
}
