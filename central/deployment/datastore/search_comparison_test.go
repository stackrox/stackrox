//go:build sql_integration

package datastore

import (
	"context"
	"sort"
	"testing"

	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

func TestSearchComparison(t *testing.T) {
	suite.Run(t, new(SearchComparisonTestSuite))
}

type SearchComparisonTestSuite struct {
	suite.Suite

	mockCtrl            *gomock.Controller
	testDB              *pgtest.TestPostgres
	imageDatastore      imageDataStore.DataStore
	deploymentDatastore DataStore
	optionsMap          search.OptionsMap
}

func (s *SearchComparisonTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())

	deploymentDS, err := GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
	s.deploymentDatastore = deploymentDS

	imageDS := imageDataStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.imageDatastore = imageDS
	s.optionsMap = pkgSchema.ImagesSchema.OptionsMap
}

func (s *SearchComparisonTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func compareResults(t *testing.T, matches bool, predResult *search.Result, searchResults []search.Result) {
	assert.Equal(t, matches, len(searchResults) != 0)
	imageKeyMap := map[string]string{"image.scan.components.vulns.cve": "imagecve.cve_base_info.cve", "image.scan.components.vulns.cvss": "imagecve.cvss"}

	if matches && len(searchResults) > 0 {
		for k := range predResult.Matches {
			sort.Strings(predResult.Matches[k])
			newImageKey, ok := imageKeyMap[k]
			// If the key exists
			if ok {
				sort.Strings(searchResults[0].Matches[newImageKey])
				assert.Equal(t, predResult.Matches[k], searchResults[0].Matches[newImageKey])
			} else {
				sort.Strings(searchResults[0].Matches[k])
				assert.Equal(t, predResult.Matches[k], searchResults[0].Matches[k])
			}
		}
	}
}

func (s *SearchComparisonTestSuite) TestImageSearchResults() {
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
			predicate, err := factory2.GeneratePredicate(c.query)
			require.NoError(t, err)

			predResult, matches := predicate.Evaluate(c.image)

			require.NoError(t, s.imageDatastore.UpsertImage(ctx, c.image))
			searchResults, err := s.imageDatastore.Search(ctx, c.query)
			require.NoError(t, err)

			compareResults(t, matches, predResult, searchResults)
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
