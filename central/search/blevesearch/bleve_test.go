package blevesearch

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/stretchr/testify/assert"
)

func TestSimpleQueries(t *testing.T) {
	indexMapping := getIndexMapping()

	index, err := bleve.NewMemOnly(indexMapping)
	assert.NoError(t, err)

	policy := &v1.Policy{
		Name: "this rocks",
	}

	err = index.Index("p", &policyWrapper{Type: v1.SearchCategory_POLICIES.String(), Policy: policy})
	assert.NoError(t, err)

	prefixQuery := bleve.NewPrefixQuery("this rocks")
	prefixQuery.SetField("policy.name")

	result, err := index.Search(bleve.NewSearchRequest(prefixQuery))
	assert.NoError(t, err)
	assert.Len(t, result.Hits, 1)
}
