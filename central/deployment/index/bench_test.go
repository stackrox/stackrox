package index

import (
	"fmt"
	"testing"

	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/require"
)

func getDeploymentIndex(b *testing.B) Indexer {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	if err != nil {
		b.Fatal(err)
	}
	return New(tmpIndex, tmpIndex)
}

func benchmarkAddDeploymentNumThen1(b *testing.B, numDeployments int) {
	indexer := getDeploymentIndex(b)
	deployment := fixtures.GetDeployment()
	addDeployments(b, indexer, deployment, numDeployments)
	deployment.Id = fmt.Sprintf("%d", numDeployments+1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.NoError(b, indexer.AddDeployment(deployment))
	}
}

func addDeployments(b *testing.B, indexer Indexer, deployment *storage.Deployment, numDeployments int) {
	for i := 0; i < numDeployments; i++ {
		deployment.Id = fmt.Sprintf("%d", i)
		require.NoError(b, indexer.AddDeployment(deployment))
	}
}

func benchmarkAddDeployment(b *testing.B, numDeployments int) {
	indexer := getDeploymentIndex(b)
	deployment := fixtures.GetDeployment()
	for i := 0; i < b.N; i++ {
		addDeployments(b, indexer, deployment, numDeployments)
	}
}

func BenchmarkAddDeployments(b *testing.B) {
	for i := 1; i <= 1000; i *= 10 {
		b.Run(fmt.Sprintf("Add Deployments - %d", i), func(subB *testing.B) {
			benchmarkAddDeployment(subB, i)
		})
	}
}

func BenchmarkAddDeploymentsThen1(b *testing.B) {
	for i := 10; i <= 1000; i *= 10 {
		b.Run(fmt.Sprintf("Add Deployments %d then 1", i), func(subB *testing.B) {
			benchmarkAddDeploymentNumThen1(subB, i)
		})
	}
}

func BenchmarkSearchDeployment(b *testing.B) {
	indexer := getDeploymentIndex(b)
	qb := search.NewQueryBuilder().AddStrings(search.Cluster, "prod cluster")
	for i := 0; i < b.N; i++ {
		_, err := indexer.Search(qb.ProtoQuery())
		require.NoError(b, err)
	}
}
