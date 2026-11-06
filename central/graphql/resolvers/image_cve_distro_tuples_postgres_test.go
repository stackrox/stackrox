//go:build sql_integration

package resolvers

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	"github.com/stackrox/rox/central/views/imagecomponentflat"
	"github.com/stackrox/rox/central/views/imagecve"
	"github.com/stackrox/rox/central/views/imagecveflat"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	// distroTuplesSameImageCVE is fixable in one component of image A and not fixable in another.
	distroTuplesSameImageCVE = "cve-2030-1"
	// distroTuplesCrossNamespaceCVE is fixable in image A, which only runs in namespace 1, and not
	// fixable in image B, which only runs in namespace 2.
	distroTuplesCrossNamespaceCVE = "cve-2030-2"

	distroTuplesAdvisory = "RHSA-2030:0001"
)

func TestImageCVEDistroTuples(t *testing.T) {
	suite.Run(t, new(ImageCVEDistroTuplesTestSuite))
}

// ImageCVEDistroTuplesTestSuite covers image CVE resolvers that represent a single CVE occurrence,
// such as ImageCVECore.distroTuples and topImageVulnerability, against Postgres-backed datastores.
type ImageCVEDistroTuplesTestSuite struct {
	suite.Suite

	testDB   *pgtest.TestPostgres
	resolver *Resolver
	schema   *graphql.Schema
}

func distroTuplesTestImages() []*storage.Image {
	return []*storage.Image{
		{
			Id: "sha256:distro-tuples-image-a",
			Name: &storage.ImageName{
				Registry: "reg-distro-tuples",
				Remote:   "image-a",
				Tag:      "1.0",
				FullName: "reg-distro-tuples/image-a:1.0",
			},
			Scan: &storage.ImageScan{
				OperatingSystem: "os-a",
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp-a1",
						Version: "1.0",
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve:        distroTuplesSameImageCVE,
								Cvss:       9.8,
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.1"},
								Advisory:   &storage.Advisory{Name: distroTuplesAdvisory, Link: "https://example.com/advisory"},
							},
							{
								Cve:        distroTuplesCrossNamespaceCVE,
								Cvss:       5.0,
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.2"},
							},
						},
					},
					{
						Name:    "comp-a2",
						Version: "2.0",
						Vulns: []*storage.EmbeddedVulnerability{
							// A lower CVSS than in comp-a1 makes the fixable comp-a1 occurrence the top
							// vulnerability of image A.
							{Cve: distroTuplesSameImageCVE, Cvss: 7.5},
						},
					},
				},
			},
		},
		{
			Id: "sha256:distro-tuples-image-b",
			Name: &storage.ImageName{
				Registry: "reg-distro-tuples",
				Remote:   "image-b",
				Tag:      "1.0",
				FullName: "reg-distro-tuples/image-b:1.0",
			},
			Scan: &storage.ImageScan{
				OperatingSystem: "os-b",
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "comp-b1",
						Version: "1.0",
						Vulns: []*storage.EmbeddedVulnerability{
							{Cve: distroTuplesCrossNamespaceCVE, Cvss: 5.0},
						},
					},
				},
			},
		},
	}
}

// distroTuplesTestDeployments runs image A twice in cluster 1 / namespace 1 and image B once in
// cluster 2 / namespace 2.
func distroTuplesTestDeployments(images []*storage.Image) []*storage.Deployment {
	deployment := func(id, name, clusterID, namespace, namespaceID string, image *storage.Image) *storage.Deployment {
		return &storage.Deployment{
			Id:          id,
			Name:        name,
			ClusterId:   clusterID,
			Namespace:   namespace,
			NamespaceId: namespaceID,
			Containers: []*storage.Container{
				{Name: "container", Image: imageTypes.ToContainerImage(image)},
			},
		}
	}
	return []*storage.Deployment{
		deployment(fixtureconsts.Deployment1, "dep-a1", fixtureconsts.Cluster1, namespace1name, fixtureconsts.Namespace1, images[0]),
		deployment(fixtureconsts.Deployment2, "dep-a2", fixtureconsts.Cluster1, namespace1name, fixtureconsts.Namespace1, images[0]),
		deployment(fixtureconsts.Deployment3, "dep-b", fixtureconsts.Cluster2, namespace2name, fixtureconsts.Namespace2, images[1]),
	}
}

func (s *ImageCVEDistroTuplesTestSuite) SetupSuite() {
	// TODO(ROX-30117): Remove when FlattenImageData feature flag is removed.
	if !features.FlattenImageData.Enabled() {
		s.T().Skip("requires the flattened image data model")
	}

	ctx := sac.WithAllAccess(context.Background())
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())

	imageDataStore := CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl)
	s.resolver, s.schema = SetupTestResolver(s.T(),
		CreateTestDeploymentDatastoreWithImageV2(s.T(), s.testDB, mockCtrl, imageDataStore),
		deploymentsView.NewDeploymentView(s.testDB.DB),
		imagesView.NewImageView(s.testDB.DB),
		imageDataStore,
		CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
		CreateTestImageCVEV2Datastore(s.T(), s.testDB),
		imagecve.NewCVEView(s.testDB.DB),
		imagecveflat.NewCVEFlatView(s.testDB.DB),
		imagecomponentflat.NewComponentFlatView(s.testDB.DB),
	)

	images := distroTuplesTestImages()
	for _, deployment := range distroTuplesTestDeployments(images) {
		s.Require().NoError(s.resolver.DeploymentDataStore.UpsertDeployment(ctx, deployment))
	}
	for _, image := range images {
		s.Require().NoError(s.resolver.ImageV2DataStore.UpsertImage(ctx, imageUtils.ConvertToV2(image)))
	}
}

// unrestrictedCtx returns a request context with a fresh loader cache and full access.
func (s *ImageCVEDistroTuplesTestSuite) unrestrictedCtx() context.Context {
	return SetAuthorizerOverride(loaders.WithLoaderContext(sac.WithAllAccess(context.Background())), allow.Anonymous())
}

// restrictedCtx returns a request context with a fresh loader cache that can only read images and
// deployments in cluster 2 / namespace 2.
func (s *ImageCVEDistroTuplesTestSuite) restrictedCtx() context.Context {
	return SetAuthorizerOverride(loaders.WithLoaderContext(sacAllowOnlyCluster2Namespace2(context.Background())), allow.Anonymous())
}

// exec runs the query through the GraphQL engine and requires that it returns no errors.
func (s *ImageCVEDistroTuplesTestSuite) exec(ctx context.Context, query string, variables map[string]interface{}, result interface{}) {
	s.T().Helper()
	response := s.schema.Exec(ctx, query, "", variables)
	s.Require().Empty(response.Errors)
	s.Require().NoError(json.Unmarshal(response.Data, result))
}

type distroTuple struct {
	IsFixable bool    `json:"isFixable"`
	EnvImpact float64 `json:"envImpact"`
}

type imageCVEWithTuples struct {
	CVE          string        `json:"cve"`
	DistroTuples []distroTuple `json:"distroTuples"`
}

func sortedFixability(tuples []distroTuple) []bool {
	fixable := make([]bool, 0, len(tuples))
	for _, tuple := range tuples {
		fixable = append(fixable, tuple.IsFixable)
	}
	sort.Slice(fixable, func(i, j int) bool { return !fixable[i] && fixable[j] })
	return fixable
}

func sortedEnvImpact(tuples []distroTuple) []float64 {
	envImpact := make([]float64, 0, len(tuples))
	for _, tuple := range tuples {
		envImpact = append(envImpact, tuple.EnvImpact)
	}
	slices.Sort(envImpact)
	return envImpact
}

func (s *ImageCVEDistroTuplesTestSuite) TestImageCVEsDistroTuplesIsFixable() {
	var data struct {
		ImageCVEs []imageCVEWithTuples `json:"imageCVEs"`
	}
	s.exec(s.unrestrictedCtx(), `query getImageCVEList($query: String, $pagination: Pagination) {
		imageCVEs(query: $query, pagination: $pagination) {
			cve
			distroTuples {
				isFixable
			}
		}
	}`, map[string]interface{}{"query": "", "pagination": map[string]interface{}{"limit": 20, "offset": 0}}, &data)

	actual := make(map[string][]bool, len(data.ImageCVEs))
	for _, cve := range data.ImageCVEs {
		actual[cve.CVE] = sortedFixability(cve.DistroTuples)
	}
	s.Equal(map[string][]bool{
		distroTuplesSameImageCVE:      {false, true},
		distroTuplesCrossNamespaceCVE: {false, true},
	}, actual)
}

func (s *ImageCVEDistroTuplesTestSuite) TestDistroTuplesReturnOccurrenceFixability() {
	ctx := s.unrestrictedCtx()
	core, err := s.resolver.ImageCVE(ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{Cve: new(distroTuplesSameImageCVE)})
	s.Require().NoError(err)
	s.Require().NotNil(core)

	tuples, err := core.DistroTuples(ctx)
	s.Require().NoError(err)
	s.Require().Len(tuples, 2)
	fixable := make([]bool, 0, len(tuples))
	for _, tuple := range tuples {
		occurrence := tuple.(*imageCVEV2Resolver).data
		isFixable, err := tuple.IsFixable(ctx, RawQuery{})
		s.Require().NoError(err)
		s.Equal(occurrence.GetFixedBy() != "", isFixable, "occurrence %s", occurrence.GetId())
		fixable = append(fixable, isFixable)
	}
	s.ElementsMatch([]bool{true, false}, fixable)
}

func (s *ImageCVEDistroTuplesTestSuite) TestImageCVEDistroTuplesIsFixableAndEnvImpact() {
	query := `query imageCVE($cve: String) {
		imageCVE(cve: $cve) {
			cve
			distroTuples {
				isFixable
				envImpact
			}
		}
	}`
	for name, tc := range map[string]struct {
		cve               string
		expectedFixable   []bool
		expectedEnvImpact []float64
	}{
		// Both occurrences are in image A, which runs in 2 of the 3 deployments.
		"same image": {
			cve:               distroTuplesSameImageCVE,
			expectedFixable:   []bool{false, true},
			expectedEnvImpact: []float64{2.0 / 3, 2.0 / 3},
		},
		// Image B runs in 1 of the 3 deployments, image A in 2.
		"cross namespace": {
			cve:               distroTuplesCrossNamespaceCVE,
			expectedFixable:   []bool{false, true},
			expectedEnvImpact: []float64{1.0 / 3, 2.0 / 3},
		},
	} {
		s.Run(name, func() {
			var data struct {
				ImageCVE imageCVEWithTuples `json:"imageCVE"`
			}
			s.exec(s.unrestrictedCtx(), query, map[string]interface{}{"cve": tc.cve}, &data)
			s.Equal(tc.cve, data.ImageCVE.CVE)
			s.Equal(tc.expectedFixable, sortedFixability(data.ImageCVE.DistroTuples))
			s.InDeltaSlice(tc.expectedEnvImpact, sortedEnvImpact(data.ImageCVE.DistroTuples), 1e-9)
		})
	}
}

func (s *ImageCVEDistroTuplesTestSuite) TestGroupedImageVulnerabilityIsFixableIsUnchanged() {
	var data struct {
		ImageVulnerabilities []struct {
			CVE       string `json:"cve"`
			IsFixable bool   `json:"isFixable"`
		} `json:"imageVulnerabilities"`
	}
	s.exec(s.unrestrictedCtx(), `query grouped($query: String) {
		imageVulnerabilities(query: $query) {
			cve
			isFixable
		}
	}`, map[string]interface{}{"query": "CVE:" + distroTuplesSameImageCVE}, &data)

	s.Require().Len(data.ImageVulnerabilities, 1)
	s.Equal(distroTuplesSameImageCVE, data.ImageVulnerabilities[0].CVE)
	// The grouped resolver reports whether any occurrence of the CVE is fixable.
	s.True(data.ImageVulnerabilities[0].IsFixable)
}

func (s *ImageCVEDistroTuplesTestSuite) TestGroupedImageVulnerabilityScopedAggregates() {
	type groupedVulnerability struct {
		CVE       string  `json:"cve"`
		IsFixable bool    `json:"isFixable"`
		EnvImpact float64 `json:"envImpact"`
	}
	var data struct {
		ImageVulnerabilities []groupedVulnerability `json:"imageVulnerabilities"`
		Images               []struct {
			Name struct {
				FullName string `json:"fullName"`
			} `json:"name"`
			ImageVulnerabilities []groupedVulnerability `json:"imageVulnerabilities"`
		} `json:"images"`
	}
	s.exec(s.unrestrictedCtx(), `query groupedAggregates {
		imageVulnerabilities {
			cve
			isFixable
			envImpact
		}
		images {
			name {
				fullName
			}
			imageVulnerabilities {
				cve
				isFixable
				envImpact
			}
		}
	}`, nil, &data)

	type aggregate struct {
		isFixable bool
		envImpact float64
	}
	toAggregates := func(vulnerabilities []groupedVulnerability) map[string]aggregate {
		aggregates := make(map[string]aggregate, len(vulnerabilities))
		for _, vulnerability := range vulnerabilities {
			aggregates[vulnerability.CVE] = aggregate{isFixable: vulnerability.IsFixable, envImpact: vulnerability.EnvImpact}
		}
		return aggregates
	}
	assertAggregates := func(scope string, expected, actual map[string]aggregate) {
		s.Require().Len(actual, len(expected), scope)
		for cve, expectedAggregate := range expected {
			s.Equal(expectedAggregate.isFixable, actual[cve].isFixable, "%s %s isFixable", scope, cve)
			s.InDelta(expectedAggregate.envImpact, actual[cve].envImpact, 1e-9, "%s %s envImpact", scope, cve)
		}
	}

	// Without a scope, each CVE aggregates all of its occurrences.
	assertAggregates("global", map[string]aggregate{
		distroTuplesSameImageCVE:      {isFixable: true, envImpact: 2.0 / 3},
		distroTuplesCrossNamespaceCVE: {isFixable: true, envImpact: 1},
	}, toAggregates(data.ImageVulnerabilities))

	// Within an image scope, each CVE aggregates only that image's occurrences.
	s.Require().Len(data.Images, 2)
	expectedByImage := map[string]map[string]aggregate{
		"reg-distro-tuples/image-a:1.0": {
			distroTuplesSameImageCVE:      {isFixable: true, envImpact: 2.0 / 3},
			distroTuplesCrossNamespaceCVE: {isFixable: true, envImpact: 2.0 / 3},
		},
		"reg-distro-tuples/image-b:1.0": {
			distroTuplesCrossNamespaceCVE: {isFixable: false, envImpact: 1.0 / 3},
		},
	}
	for _, image := range data.Images {
		expected, ok := expectedByImage[image.Name.FullName]
		s.Require().True(ok, image.Name.FullName)
		assertAggregates(image.Name.FullName, expected, toAggregates(image.ImageVulnerabilities))
	}
}

func (s *ImageCVEDistroTuplesTestSuite) TestTopImageVulnerabilityFields() {
	var data struct {
		Images []struct {
			Name struct {
				FullName string `json:"fullName"`
			} `json:"name"`
			TopImageVulnerability struct {
				CVE       string `json:"cve"`
				IsFixable bool   `json:"isFixable"`
			} `json:"topImageVulnerability"`
		} `json:"images"`
		ImageComponents []struct {
			Name                  string `json:"name"`
			TopImageVulnerability struct {
				CVE            string `json:"cve"`
				FixedByVersion string `json:"fixedByVersion"`
				Advisory       *struct {
					Name string `json:"name"`
				} `json:"advisory"`
				IsFixable bool `json:"isFixable"`
			} `json:"topImageVulnerability"`
		} `json:"imageComponents"`
	}
	s.exec(s.unrestrictedCtx(), `query topVulnerabilities {
		images {
			name {
				fullName
			}
			topImageVulnerability {
				cve
				isFixable
			}
		}
		imageComponents {
			name
			topImageVulnerability {
				cve
				fixedByVersion
				advisory {
					name
				}
				isFixable
			}
		}
	}`, nil, &data)

	type imageTopVulnerability struct {
		cve       string
		isFixable bool
	}
	actualImages := make(map[string]imageTopVulnerability, len(data.Images))
	for _, image := range data.Images {
		actualImages[image.Name.FullName] = imageTopVulnerability{
			cve:       image.TopImageVulnerability.CVE,
			isFixable: image.TopImageVulnerability.IsFixable,
		}
	}
	s.Equal(map[string]imageTopVulnerability{
		"reg-distro-tuples/image-a:1.0": {cve: distroTuplesSameImageCVE, isFixable: true},
		"reg-distro-tuples/image-b:1.0": {cve: distroTuplesCrossNamespaceCVE, isFixable: false},
	}, actualImages)

	type componentTopVulnerability struct {
		cve            string
		fixedByVersion string
		advisory       string
		isFixable      bool
	}
	actual := make(map[string]componentTopVulnerability, len(data.ImageComponents))
	for _, component := range data.ImageComponents {
		top := component.TopImageVulnerability
		advisory := ""
		if top.Advisory != nil {
			advisory = top.Advisory.Name
		}
		actual[component.Name] = componentTopVulnerability{
			cve:            top.CVE,
			fixedByVersion: top.FixedByVersion,
			advisory:       advisory,
			isFixable:      top.IsFixable,
		}
	}
	s.Equal(map[string]componentTopVulnerability{
		"comp-a1": {cve: distroTuplesSameImageCVE, fixedByVersion: "1.1", advisory: distroTuplesAdvisory, isFixable: true},
		"comp-a2": {cve: distroTuplesSameImageCVE},
		"comp-b1": {cve: distroTuplesCrossNamespaceCVE},
	}, actual)
}

func (s *ImageCVEDistroTuplesTestSuite) TestRestrictedAccess() {
	query := `query restricted($cve: String, $query: String) {
		imageCVE(cve: $cve) {
			cve
			distroTuples {
				isFixable
				envImpact
			}
		}
		imageVulnerabilities(query: $query) {
			cve
			isFixable
		}
	}`
	variables := map[string]interface{}{
		"cve":   distroTuplesCrossNamespaceCVE,
		"query": "CVE:" + distroTuplesCrossNamespaceCVE,
	}
	for name, tc := range map[string]struct {
		ctx               context.Context
		expectedFixable   []bool
		expectedEnvImpact []float64
		expectedGrouped   bool
	}{
		"unrestricted access sees both occurrences": {
			ctx:               s.unrestrictedCtx(),
			expectedFixable:   []bool{false, true},
			expectedEnvImpact: []float64{1.0 / 3, 2.0 / 3},
			expectedGrouped:   true,
		},
		// The fixable occurrence in namespace 1 must not be visible or affect any value.
		"namespace 2 access sees only the non-fixable occurrence": {
			ctx:               s.restrictedCtx(),
			expectedFixable:   []bool{false},
			expectedEnvImpact: []float64{1},
			expectedGrouped:   false,
		},
	} {
		s.Run(name, func() {
			var data struct {
				ImageCVE             imageCVEWithTuples `json:"imageCVE"`
				ImageVulnerabilities []struct {
					CVE       string `json:"cve"`
					IsFixable bool   `json:"isFixable"`
				} `json:"imageVulnerabilities"`
			}
			s.exec(tc.ctx, query, variables, &data)
			s.Equal(distroTuplesCrossNamespaceCVE, data.ImageCVE.CVE)
			s.Equal(tc.expectedFixable, sortedFixability(data.ImageCVE.DistroTuples))
			s.InDeltaSlice(tc.expectedEnvImpact, sortedEnvImpact(data.ImageCVE.DistroTuples), 1e-9)
			s.Require().Len(data.ImageVulnerabilities, 1)
			s.Equal(tc.expectedGrouped, data.ImageVulnerabilities[0].IsFixable)
		})
	}
}
