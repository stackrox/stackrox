package blevesearch

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stretchr/testify/assert"
)

func TestSimpleQueries(t *testing.T) {
	indexMapping := getIndexMapping()
	index, err := bleve.NewMemOnly(indexMapping)
	assert.NoError(t, err)

	err = index.Index("test", map[string]string{
		"name": "this rocks",
	})
	assert.NoError(t, err)

	result, err := index.Search(bleve.NewSearchRequest(bleve.NewPrefixQuery("this rocks")))
	assert.NoError(t, err)
	assert.Len(t, result.Hits, 1)
}
