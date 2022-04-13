package index

import (
	"sort"
	"testing"

	"github.com/stackrox/stackrox/central/globalindex"
	imageIndex "github.com/stackrox/stackrox/central/image/index"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/predicate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compareResults(t *testing.T, matches bool, predResult *search.Result, searchResults []search.Result) {
	assert.Equal(t, matches, len(searchResults) != 0)
	if matches && len(searchResults) > 0 {
		for k := range predResult.Matches {
			sort.Strings(predResult.Matches[k])
			sort.Strings(searchResults[0].Matches[k])
			assert.Equal(t, predResult.Matches[k], searchResults[0].Matches[k])
		}
	}
}

func TestImageSearchResults(t *testing.T) {
	cases := []struct {
		image *storage.Image
		query *v1.Query
	}{
		{
			image: fixtures.GetImage(),
			query: search.NewQueryBuilder().AddStringsHighlighted(search.ImageTag, "latest").ProtoQuery(),
		},
		{
			image: fixtures.GetImage(),
			query: search.NewQueryBuilder().AddLinkedFieldsHighlighted(
				[]search.FieldLabel{search.CVSS, search.CVE},
				[]string{">=5", search.WildcardString}).
				ProtoQuery(),
		},
		{
			image: fixtures.GetImage(),
			query: search.NewQueryBuilder().AddLinkedFieldsHighlighted(
				[]search.FieldLabel{search.CVSS, search.CVE},
				[]string{">4", "CVE-2014-620"}).
				ProtoQuery(),
		},
	}

	idx, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	index := imageIndex.New(idx)

	factory := predicate.NewFactory("image", (*storage.Image)(nil))
	for _, c := range cases {
		t.Run("test", func(t *testing.T) {
			predicate, err := factory.GeneratePredicate(c.query)
			require.NoError(t, err)

			predResult, matches := predicate.Evaluate(c.image)

			require.NoError(t, index.AddImage(c.image))
			searchResults, err := index.Search(c.query)
			require.NoError(t, err)

			compareResults(t, matches, predResult, searchResults)
		})
	}
}

func TestDeploymentSearchResults(t *testing.T) {
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

	idx, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	index := New(idx, idx)

	factory := predicate.NewFactory("deployment", (*storage.Deployment)(nil))
	for _, c := range cases {
		t.Run("test", func(t *testing.T) {
			predicate, err := factory.GeneratePredicate(c.query)
			require.NoError(t, err)

			predResult, matches := predicate.Evaluate(c.deployment)

			require.NoError(t, index.AddDeployment(c.deployment))
			searchResults, err := index.Search(c.query)
			require.NoError(t, err)

			compareResults(t, matches, predResult, searchResults)
		})
	}
}
