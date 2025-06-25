//go:build sql_integration

package deployments

import (
	"context"
	"sort"
	"testing"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imagePostgresV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	imageSamples "github.com/stackrox/rox/pkg/fixtures/image"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type testCase struct {
	desc                    string
	ctx                     context.Context
	query                   *v1.Query
	matchFilter             *filterImpl
	less                    lessFunc
	isError                 bool
	hasSortBySeverityCounts bool
	ignoreOrder             bool
}

type lessFunc func(records []*deploymentResponse) func(i, j int) bool

type filterImpl struct {
	matchDeployment func(dep *storage.Deployment) bool
	matchImage      func(image *storage.Image) bool
	matchVuln       func(vuln *storage.EmbeddedVulnerability) bool
}

func matchAllFilter() *filterImpl {
	return &filterImpl{
		matchDeployment: func(dep *storage.Deployment) bool {
			return true
		},
		matchImage: func(_ *storage.Image) bool {
			return true
		},
		matchVuln: func(_ *storage.EmbeddedVulnerability) bool {
			return true
		},
	}
}

func (f *filterImpl) withDeploymentFilter(fn func(dep *storage.Deployment) bool) *filterImpl {
	f.matchDeployment = fn
	return f
}

func (f *filterImpl) withImageFilter(fn func(image *storage.Image) bool) *filterImpl {
	f.matchImage = fn
	return f
}

func (f *filterImpl) withVulnFilter(fn func(vuln *storage.EmbeddedVulnerability) bool) *filterImpl {
	f.matchVuln = fn
	return f
}

func TestDeploymentView(t *testing.T) {
	if !features.FlattenCVEData.Enabled() {
		t.Skip("FlattenCVEData is disabled")
	}
	suite.Run(t, new(DeploymentViewTestSuite))
}

type DeploymentViewTestSuite struct {
	suite.Suite

	testDB         *pgtest.TestPostgres
	deploymentView DeploymentView

	testDeployments      []*storage.Deployment
	testDeploymentsMap   map[string]*storage.Deployment
	testImages           []*storage.Image
	scopeToDeploymentIDs map[string]set.StringSet
}

func (s *DeploymentViewTestSuite) SetupSuite() {
	mockCtrl := gomock.NewController(s.T())
	ctx := sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	mockRisk := mockRisks.NewMockDataStore(mockCtrl)

	// Initialize the datastore.
	imageStore := imageDS.NewWithPostgres(
		imagePostgresV2.New(s.testDB.DB, false, concurrency.NewKeyFence()),
		mockRisk,
		ranking.ImageRanker(),
		ranking.ComponentRanker(),
	)
	deploymentStore, err := deploymentDS.NewTestDataStore(
		s.T(),
		s.testDB,
		&deploymentDS.DeploymentTestStoreParams{
			ImagesDataStore:  imageStore,
			RisksDataStore:   mockRisk,
			ClusterRanker:    ranking.ClusterRanker(),
			NamespaceRanker:  ranking.NamespaceRanker(),
			DeploymentRanker: ranking.DeploymentRanker(),
		},
	)
	s.Require().NoError(err)

	// Upsert test images.
	images, err := imageSamples.GetTestImages(s.T())
	s.Require().NoError(err)
	// set cvss metrics list with one nvd cvss score
	for _, image := range images {
		s.Require().NoError(imageStore.UpsertImage(ctx, image))
	}

	// Ensure that the image is stored and constructed as expected.
	for idx, image := range images {
		actual, found, err := imageStore.GetImage(ctx, image.GetId())
		s.Require().NoError(err)
		s.Require().True(found)

		cloned := actual.CloneVT()
		// Adjust dynamic fields and ensure images in ACS are as expected.
		standardizeImages(image, cloned)

		// Now that we confirmed that images match, use stored image to establish the expected test results.
		// This makes dynamic fields matching (e.g. created at) straightforward.
		images[idx] = actual
	}
	s.testImages = images
	s.deploymentView = NewDeploymentView(s.testDB.DB)

	s.Require().Len(images, 5)
	deployments := []*storage.Deployment{
		fixtures.GetDeploymentWithImage(testconsts.Cluster1, testconsts.NamespaceA, images[0]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster1, testconsts.NamespaceA, images[1]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, images[2]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, images[3]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster3, testconsts.NamespaceC, images[4]),
	}

	s.testDeploymentsMap = make(map[string]*storage.Deployment)
	for _, d := range deployments {
		s.Require().NoError(deploymentStore.UpsertDeployment(ctx, d))
		s.testDeploymentsMap[d.GetId()] = d
	}
	s.testDeployments = deployments

	s.scopeToDeploymentIDs = map[string]set.StringSet{
		testutils.UnrestrictedReadWriteCtx: set.NewStringSet(
			deployments[0].GetId(),
			deployments[1].GetId(),
			deployments[2].GetId(),
			deployments[3].GetId(),
			deployments[4].GetId(),
		),
		testutils.Cluster1ReadWriteCtx:           set.NewStringSet(deployments[0].GetId(), deployments[1].GetId()),
		testutils.Cluster1NamespaceAReadWriteCtx: set.NewStringSet(deployments[0].GetId(), deployments[1].GetId()),
		testutils.Cluster2ReadWriteCtx:           set.NewStringSet(deployments[2].GetId(), deployments[3].GetId()),
		testutils.Cluster2NamespaceBReadWriteCtx: set.NewStringSet(deployments[2].GetId(), deployments[3].GetId()),
		testutils.Cluster3ReadWriteCtx:           set.NewStringSet(deployments[4].GetId()),
		testutils.Cluster3NamespaceCReadWriteCtx: set.NewStringSet(deployments[4].GetId()),
	}
}

func (s *DeploymentViewTestSuite) TestGet() {
	contextMap := testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Deployment)
	for _, tc := range []testCase{
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
			ctx:         contextMap[testutils.UnrestrictedReadWriteCtx],
			desc:        "filtered query",
			query:       search.NewQueryBuilder().AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:latest").ProtoQuery(),
			ignoreOrder: true,
			matchFilter: matchAllFilter().withImageFilter(func(image *storage.Image) bool {
				return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:latest"
			}),
		},
		{
			ctx:  contextMap[testutils.UnrestrictedReadWriteCtx],
			desc: "order by risk score",
			query: search.NewQueryBuilder().
				WithPagination(
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.DeploymentRiskScore).Reversed(true)).
						AddSortOption(search.NewSortOption(search.DeploymentID)),
				).
				ProtoQuery(),
			matchFilter: matchAllFilter(),
			less: func(records []*deploymentResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					scorei := s.testDeploymentsMap[records[i].DeploymentID].RiskScore
					scorej := s.testDeploymentsMap[records[j].DeploymentID].RiskScore
					if scorei == scorej {
						return records[i].DeploymentID < records[j].DeploymentID
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
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.DeploymentRiskScore).Reversed(true)).
						AddSortOption(search.NewSortOption(search.DeploymentID)),
				).
				ProtoQuery(),
			matchFilter: matchAllFilter().withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
				return vuln.GetFixedBy() != ""
			}),
			less: func(records []*deploymentResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					scorei := s.testDeploymentsMap[records[i].DeploymentID].RiskScore
					scorej := s.testDeploymentsMap[records[j].DeploymentID].RiskScore
					if scorei == scorej {
						return records[i].DeploymentID < records[j].DeploymentID
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
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.CriticalSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.DeploymentID)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter:             matchAllFilter(),
			less: func(records []*deploymentResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if records[i].CriticalSeverityCount == records[j].CriticalSeverityCount {
						return records[i].DeploymentID < records[j].DeploymentID
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
						AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.DeploymentID)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter:             matchAllFilter(),
			less: func(records []*deploymentResponse) func(i int, j int) bool {
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
					return records[i].DeploymentID < records[j].DeploymentID
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
						AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.DeploymentID)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter: matchAllFilter().withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
				return vuln.Severity == storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY ||
					vuln.Severity == storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
			}),
			less: func(records []*deploymentResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if !(records[i].ModerateSeverityCount == records[j].ModerateSeverityCount) {
						return records[i].ModerateSeverityCount > records[j].ModerateSeverityCount
					}
					if !(records[i].LowSeverityCount == records[j].LowSeverityCount) {
						return records[i].LowSeverityCount > records[j].LowSeverityCount
					}
					return records[i].DeploymentID < records[j].DeploymentID
				}
			},
		},
		{
			ctx:         contextMap[testutils.Cluster1ReadWriteCtx],
			desc:        "cluster-1 scoped context",
			query:       search.EmptyQuery(),
			ignoreOrder: true,
			matchFilter: matchAllFilter().withDeploymentFilter(func(dep *storage.Deployment) bool {
				return s.scopeToDeploymentIDs[testutils.Cluster1ReadWriteCtx].Contains(dep.GetId())
			}),
		},
		{
			ctx:         contextMap[testutils.Cluster1NamespaceAReadWriteCtx],
			desc:        "cluster-1 namespace-A scoped context",
			query:       search.EmptyQuery(),
			ignoreOrder: true,
			matchFilter: matchAllFilter().withDeploymentFilter(func(dep *storage.Deployment) bool {
				return s.scopeToDeploymentIDs[testutils.Cluster1NamespaceAReadWriteCtx].Contains(dep.GetId())
			}),
		},
		{
			ctx:  contextMap[testutils.Cluster2ReadWriteCtx],
			desc: "cluster-2 scoped context and query + multi-sort by severity counts",
			query: search.NewQueryBuilder().
				WithPagination(
					search.NewPagination().
						AddSortOption(search.NewSortOption(search.ModerateSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.DeploymentID)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter: matchAllFilter().withDeploymentFilter(func(dep *storage.Deployment) bool {
				return s.scopeToDeploymentIDs[testutils.Cluster2ReadWriteCtx].Contains(dep.GetId())
			}),
			less: func(records []*deploymentResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if !(records[i].ModerateSeverityCount == records[j].ModerateSeverityCount) {
						return records[i].ModerateSeverityCount > records[j].ModerateSeverityCount
					}
					if !(records[i].LowSeverityCount == records[j].LowSeverityCount) {
						return records[i].LowSeverityCount > records[j].LowSeverityCount
					}
					return records[i].DeploymentID < records[j].DeploymentID
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
						AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)).
						AddSortOption(search.NewSortOption(search.DeploymentID)),
				).
				ProtoQuery(),
			hasSortBySeverityCounts: true,
			matchFilter: matchAllFilter().withDeploymentFilter(func(dep *storage.Deployment) bool {
				return s.scopeToDeploymentIDs[testutils.Cluster2NamespaceBReadWriteCtx].Contains(dep.GetId())
			}),
			less: func(records []*deploymentResponse) func(i int, j int) bool {
				return func(i int, j int) bool {
					if !(records[i].ModerateSeverityCount == records[j].ModerateSeverityCount) {
						return records[i].ModerateSeverityCount > records[j].ModerateSeverityCount
					}
					if !(records[i].LowSeverityCount == records[j].LowSeverityCount) {
						return records[i].LowSeverityCount > records[j].LowSeverityCount
					}
					return records[i].DeploymentID < records[j].DeploymentID
				}
			},
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			deploymentCores, err := s.deploymentView.Get(tc.ctx, tc.query)
			if tc.isError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			expectedCores := s.compileExpected(tc.matchFilter, tc.less, tc.hasSortBySeverityCounts)
			if tc.ignoreOrder {
				assert.ElementsMatch(t, expectedCores, deploymentCores)
			} else {
				assert.Equal(t, expectedCores, deploymentCores)
			}
		})
	}
}

func (s *DeploymentViewTestSuite) compileExpected(filter *filterImpl, less lessFunc, hasSortBySeverityCounts bool) []DeploymentCore {
	imageMap := make(map[string]*storage.Image)
	for _, img := range s.testImages {
		imageMap[img.GetId()] = img
	}

	expected := make([]*deploymentResponse, 0)
	for _, deployment := range s.testDeployments {
		if !filter.matchDeployment(deployment) {
			continue
		}

		vulns := compileExpectedVulns(deployment.GetContainers(), imageMap, filter)
		if len(vulns) == 0 {
			continue
		}

		val := &deploymentResponse{
			DeploymentID: deployment.GetId(),
		}
		if hasSortBySeverityCounts {
			gatherSeverityAndFixabilityCounts(val, vulns)
		}

		expected = append(expected, val)
	}

	if less != nil {
		sort.SliceStable(expected, less(expected))
	}
	ret := make([]DeploymentCore, 0, len(expected))
	for _, r := range expected {
		ret = append(ret, r)
	}
	return ret
}

func compileExpectedVulns(containers []*storage.Container, imageMap map[string]*storage.Image, filter *filterImpl) []*storage.EmbeddedVulnerability {
	results := make([]*storage.EmbeddedVulnerability, 0)
	for _, container := range containers {
		image := imageMap[container.GetImage().GetId()]
		if image == nil {
			continue
		}
		if !filter.matchImage(image) {
			continue
		}

		for _, component := range image.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				if !filter.matchVuln(vuln) {
					continue
				}
				results = append(results, vuln)
			}
		}
	}
	return results
}

func gatherSeverityAndFixabilityCounts(val *deploymentResponse, vulns []*storage.EmbeddedVulnerability) {
	criticalSevCVEs := set.NewStringSet()
	criticalSevFixableCVEs := set.NewStringSet()
	importantSevCVEs := set.NewStringSet()
	importantSevFixableCVEs := set.NewStringSet()
	moderateSevCVEs := set.NewStringSet()
	moderateSevFixableCVEs := set.NewStringSet()
	lowSevCVEs := set.NewStringSet()
	lowSevFixableCVEs := set.NewStringSet()
	unknownSevCVEs := set.NewStringSet()
	unknownSevFixableCVEs := set.NewStringSet()

	for _, vuln := range vulns {
		switch vuln.GetSeverity() {
		case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
			criticalSevCVEs.Add(vuln.GetCve())
			if vuln.GetFixedBy() != "" {
				criticalSevFixableCVEs.Add(vuln.GetCve())
			}
		case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
			importantSevCVEs.Add(vuln.GetCve())
			if vuln.GetFixedBy() != "" {
				importantSevFixableCVEs.Add(vuln.GetCve())
			}
		case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
			moderateSevCVEs.Add(vuln.GetCve())
			if vuln.GetFixedBy() != "" {
				moderateSevFixableCVEs.Add(vuln.GetCve())
			}
		case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
			lowSevCVEs.Add(vuln.GetCve())
			if vuln.GetFixedBy() != "" {
				lowSevFixableCVEs.Add(vuln.GetCve())
			}
		case storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY:
			unknownSevCVEs.Add(vuln.GetCve())
			if vuln.GetFixedBy() != "" {
				unknownSevFixableCVEs.Add(vuln.GetCve())
			}
		}
	}

	val.CriticalSeverityCount = criticalSevCVEs.Cardinality()
	val.FixableCriticalSeverityCount = criticalSevFixableCVEs.Cardinality()
	val.ImportantSeverityCount = importantSevCVEs.Cardinality()
	val.FixableImportantSeverityCount = importantSevFixableCVEs.Cardinality()
	val.ModerateSeverityCount = moderateSevCVEs.Cardinality()
	val.FixableModerateSeverityCount = moderateSevFixableCVEs.Cardinality()
	val.LowSeverityCount = lowSevCVEs.Cardinality()
	val.FixableLowSeverityCount = lowSevFixableCVEs.Cardinality()
	val.UnknownSeverityCount = unknownSevCVEs.Cardinality()
	val.FixableUnknownSeverityCount = unknownSevFixableCVEs.Cardinality()
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
		}
	}
}
