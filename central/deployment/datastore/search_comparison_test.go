//go:build sql_integration

package datastore

import (
	"context"
	"slices"
	"testing"

	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	imageV2DataStore "github.com/stackrox/rox/central/imagev2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

func TestSearchComparison(t *testing.T) {
	suite.Run(t, new(SearchComparisonTestSuite))
}

type SearchComparisonTestSuite struct {
	suite.Suite

	testDB              *pgtest.TestPostgres
	imageDatastore      imageDataStore.DataStore
	imageV2Datastore    imageV2DataStore.DataStore
	deploymentDatastore DataStore
	optionsMap          search.OptionsMap
}

func (s *SearchComparisonTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())

	deploymentDS, err := GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
	s.deploymentDatastore = deploymentDS

	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	if features.FlattenImageData.Enabled() {
		imageV2DS := imageV2DataStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
		s.imageV2Datastore = imageV2DS
		s.optionsMap = pkgSchema.ImagesV2Schema.OptionsMap
	} else {
		imageDS := imageDataStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
		s.imageDatastore = imageDS
		s.optionsMap = pkgSchema.ImagesSchema.OptionsMap
	}
}

// TODO(ROX-30117): Remove when FlattenImageData feature flag is removed.
func compareResults(t *testing.T, matches bool, predResult *search.Result, searchResults []search.Result) {
	assert.Equal(t, matches, len(searchResults) != 0)
	imageKeyMap := map[string]string{"image.scan.components.vulns.cve": "imagecvev2.cve_base_info.cve", "image.scan.components.vulns.cvss": "imagecvev2.cvss"}

	if matches && len(searchResults) > 0 {
		for k := range predResult.Matches {
			slices.Sort(predResult.Matches[k])
			newImageKey, ok := imageKeyMap[k]
			// If the key exists
			if ok {
				slices.Sort(searchResults[0].Matches[newImageKey])
				assert.Equal(t, predResult.Matches[k], searchResults[0].Matches[newImageKey])
			} else {
				slices.Sort(searchResults[0].Matches[k])
				assert.Equal(t, predResult.Matches[k], searchResults[0].Matches[k])
			}
		}
	}
}

func compareResultsV2(t *testing.T, matches bool, predResult *search.Result, searchResults []search.Result) {
	assert.Equal(t, matches, len(searchResults) != 0)
	imageKeyMap := map[string]string{"imagev2.scan.components.vulns.cve": "imagecvev2.cve_base_info.cve", "imagev2.scan.components.vulns.cvss": "imagecvev2.cvss"}

	if matches && len(searchResults) > 0 {
		for k := range predResult.Matches {
			slices.Sort(predResult.Matches[k])
			newImageKey, ok := imageKeyMap[k]
			// If the key exists
			if ok {
				slices.Sort(searchResults[0].Matches[newImageKey])
				assert.Equal(t, predResult.Matches[k], searchResults[0].Matches[newImageKey])
			} else {
				slices.Sort(searchResults[0].Matches[k])
				assert.Equal(t, predResult.Matches[k], searchResults[0].Matches[k])
			}
		}
	}
}

func (s *SearchComparisonTestSuite) TestImageSearchResults() {
	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	if features.FlattenImageData.Enabled() {
		s.testImageSearchResultsV2()
	} else {
		s.testImageSearchResultsV1()
	}
}

func (s *SearchComparisonTestSuite) testImageSearchResultsV1() {
	cases := []struct {
		image          *storage.Image
		query          *v1.Query
		expectedResult *search.Result
	}{
		{
			image: fixtures.GetImage(),
			query: search.NewQueryBuilder().AddStringsHighlighted(search.ImageTag, "latest").ProtoQuery(),
			expectedResult: &search.Result{
				ID: "test",
				Matches: map[string][]string{"imagecve.cve_base_info.cve": {"CVE-2014-6200", "CVE-2014-6201", "CVE-2014-6202", "CVE-2014-6203", "CVE-2014-6204"},
					"imagecve.cvss": {"5", "5", "5", "5", "5"},
				},
			},
		},
		{
			image: fixtures.GetImageWithUniqueComponents(50),
			query: search.NewQueryBuilder().AddLinkedFieldsHighlighted(
				[]search.FieldLabel{search.CVSS, search.CVE},
				[]string{">=5", search.WildcardString}).
				ProtoQuery(),
			expectedResult: &search.Result{
				ID: "test",
				Matches: map[string][]string{"imagecve.cve_base_info.cve": {"CVE-2014-6200", "CVE-2014-6201", "CVE-2014-6202", "CVE-2014-6203", "CVE-2014-6204"},
					"imagecve.cvss": {"5", "5", "5", "5", "5"},
				},
			},
		},
		{
			image: fixtures.GetImageWithUniqueComponents(50),
			query: search.NewQueryBuilder().AddLinkedFieldsHighlighted(
				[]search.FieldLabel{search.CVSS, search.CVE},
				[]string{">2", "CVE-2014-620"}).
				ProtoQuery(),
			expectedResult: &search.Result{
				ID: "test",
				Matches: map[string][]string{"imagecve.cve_base_info.cve": {"CVE-2014-6200", "CVE-2014-6201", "CVE-2014-6202", "CVE-2014-6203", "CVE-2014-6204"},
					"imagecve.cvss": {"5", "5", "5", "5", "5"},
				},
			},
		},
	}

	factory := predicate.NewFactory("image", (*storage.Image)(nil))
	factory2 := factory.ForCustomOptionsMap(s.optionsMap)
	for _, c := range cases {
		s.T().Run("test", func(t *testing.T) {
			pred, err := factory2.GeneratePredicate(c.query)
			require.NoError(t, err)

			predResult, matches := pred.Evaluate(c.image)

			require.NoError(t, s.imageDatastore.UpsertImage(ctx, c.image))
			searchResults, err := s.imageDatastore.Search(ctx, c.query)
			require.NoError(t, err)

			compareResults(t, matches, predResult, searchResults)
		})
	}
}

func (s *SearchComparisonTestSuite) testImageSearchResultsV2() {
	cases := []struct {
		image          *storage.ImageV2
		query          *v1.Query
		expectedResult *search.Result
	}{
		{
			image: fixtures.GetImageV2(),
			query: search.NewQueryBuilder().AddStringsHighlighted(search.ImageTag, "latest").ProtoQuery(),
			expectedResult: &search.Result{
				ID: "test",
				Matches: map[string][]string{"imagecvev2.cve_base_info.cve": {"CVE-2014-6200", "CVE-2014-6201", "CVE-2014-6202", "CVE-2014-6203", "CVE-2014-6204"},
					"imagecvev2.cvss": {"5", "5", "5", "5", "5"},
				},
			},
		},
		{
			image: fixtures.GetImageV2WithUniqueComponents(50),
			query: search.NewQueryBuilder().AddLinkedFieldsHighlighted(
				[]search.FieldLabel{search.CVSS, search.CVE},
				[]string{">=5", search.WildcardString}).
				ProtoQuery(),
			expectedResult: &search.Result{
				ID: "test",
				Matches: map[string][]string{"imagecvev2.cve_base_info.cve": {"CVE-2014-6200", "CVE-2014-6201", "CVE-2014-6202", "CVE-2014-6203", "CVE-2014-6204"},
					"imagecvev2.cvss": {"5", "5", "5", "5", "5"},
				},
			},
		},
		{
			image: fixtures.GetImageV2WithUniqueComponents(50),
			query: search.NewQueryBuilder().AddLinkedFieldsHighlighted(
				[]search.FieldLabel{search.CVSS, search.CVE},
				[]string{">2", "CVE-2014-620"}).
				ProtoQuery(),
			expectedResult: &search.Result{
				ID: "test",
				Matches: map[string][]string{"imagecvev2.cve_base_info.cve": {"CVE-2014-6200", "CVE-2014-6201", "CVE-2014-6202", "CVE-2014-6203", "CVE-2014-6204"},
					"imagecvev2.cvss": {"5", "5", "5", "5", "5"},
				},
			},
		},
	}

	factory := predicate.NewFactory("imagev2", (*storage.ImageV2)(nil))
	factory2 := factory.ForCustomOptionsMap(s.optionsMap)
	for _, c := range cases {
		s.T().Run("test", func(t *testing.T) {
			pred, err := factory2.GeneratePredicate(c.query)
			require.NoError(t, err)

			predResult, matches := pred.Evaluate(c.image)

			require.NoError(t, s.imageV2Datastore.UpsertImage(ctx, c.image))
			searchResults, err := s.imageV2Datastore.Search(ctx, c.query)
			require.NoError(t, err)

			compareResultsV2(t, matches, predResult, searchResults)
		})
	}
}

func (s *SearchComparisonTestSuite) TestDeploymentSearchResults() {
	cases := []struct {
		deployment *storage.Deployment
		query      *v1.Query
	}{
		{
			deployment: fixtures.GetDeployment(),
			query:      search.NewQueryBuilder().AddStringsHighlighted(search.Cluster, "prod").ProtoQuery(),
		},
		{
			deployment: fixtures.GetDeployment(),
			query:      search.NewQueryBuilder().AddBoolsHighlighted(search.Privileged, true).ProtoQuery(),
		},
		{
			deployment: fixtures.GetDeployment(),
			query: search.NewQueryBuilder().AddGenericTypeLinkedFieldsHighligted(
				[]search.FieldLabel{search.AddCapabilities, search.Privileged}, []interface{}{"SYS_ADMIN", true}).ProtoQuery(),
		},
	}

	factory := predicate.NewFactory("deployment", (*storage.Deployment)(nil))
	for _, c := range cases {
		s.T().Run("test", func(t *testing.T) {
			predicate, err := factory.GeneratePredicate(c.query)
			require.NoError(t, err)

			predResult, matches := predicate.Evaluate(c.deployment)

			require.NoError(t, s.deploymentDatastore.UpsertDeployment(ctx, c.deployment))
			searchResults, err := s.deploymentDatastore.Search(ctx, c.query)
			require.NoError(t, err)

			compareResults(t, matches, predResult, searchResults)
		})
	}
}
