package blevesearch

import (
	"fmt"
	"math"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/index/scorch"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func preload(b *testing.B) bleve.Index {
	tmpDir := b.TempDir()

	kvconfig := map[string]interface{}{
		// Persist the index
		"unsafe_batch": false,
	}

	index, err := bleve.NewUsing(tmpDir, bleve.NewIndexMapping(), scorch.Name, scorch.Name, kvconfig)
	require.NoError(b, err)

	for i := 0; i < 100; i++ {
		batch := index.NewBatch()
		for j := 0; j < 100; j++ {
			pi := fixtures.GetProcessIndicator()
			pi.Id = uuid.NewV4().String()
			pi.DeploymentId = fmt.Sprintf("%d", i)
			require.NoError(b, batch.Index(pi.Id, pi))
		}
		require.NoError(b, index.Batch(batch))
	}
	return index
}

func run(b *testing.B, index bleve.Index, q query.Query) {
	req := bleve.NewSearchRequest(q)
	req.Size = math.MaxInt32
	for i := 0; i < b.N; i++ {
		_, err := index.Search(req)
		require.NoError(b, err)
	}
}

func BenchmarkCustomNegationQuery(b *testing.B) {
	index := preload(b)
	b.Run("actual", func(b *testing.B) {
		nq := NewNegationQuery(query.NewMatchAllQuery(), NewMatchPhrasePrefixQuery("deploymentId", "5"), false)
		run(b, index, nq)
	})
}

func BenchmarkBleveNegationQuery(b *testing.B) {
	index := preload(b)
	b.Run("actual", func(b *testing.B) {
		bq := bleve.NewBooleanQuery()
		bq.AddMustNot(NewMatchPhrasePrefixQuery("deploymentId", "5"))
		run(b, index, bq)
	})
}
