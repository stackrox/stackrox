package index

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
)

func getAlertIndex() Indexer {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	if err != nil {
		panic(err)
	}
	return New(tmpIndex)
}

func benchmarkAddAlertNumThen1(b *testing.B, numAlerts int) {
	indexer := getAlertIndex()
	alert := fixtures.GetAlert()
	addAlerts(b, indexer, alert, numAlerts)
	alert.Id = fmt.Sprintf("%d", numAlerts+1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.NoError(b, indexer.AddAlert(alert))
	}
}

func addAlerts(b *testing.B, indexer Indexer, alert *storage.Alert, numAlerts int) {
	for i := 0; i < numAlerts; i++ {
		alert.Id = fmt.Sprintf("%d", i)
		require.NoError(b, indexer.AddAlert(alert))
	}
}

func benchmarkAddAlert(b *testing.B, numAlerts int) {
	indexer := getAlertIndex()
	alert := fixtures.GetAlert()
	for i := 0; i < b.N; i++ {
		addAlerts(b, indexer, alert, numAlerts)
	}
}

func BenchmarkAddAlerts(b *testing.B) {
	for i := 1; i <= 1000; i *= 10 {
		b.Run(fmt.Sprintf("Add Alerts - %d", i), func(subB *testing.B) {
			benchmarkAddAlert(subB, i)
		})
	}
}

func BenchmarkAddAlertsThen1(b *testing.B) {
	for i := 10; i <= 1000; i *= 10 {
		b.Run(fmt.Sprintf("Add Alerts %d then 1", i), func(subB *testing.B) {
			benchmarkAddAlertNumThen1(subB, i)
		})
	}
}

func BenchmarkSearchAlert(b *testing.B) {
	indexer := getAlertIndex()
	qb := search.NewQueryBuilder().AddStrings(search.Cluster, "prod cluster")
	for i := 0; i < b.N; i++ {
		_, err := indexer.Search(qb.ProtoQuery())
		require.NoError(b, err)
	}
}

func BenchmarkBatch(b *testing.B) {
	indexer := getAlertIndex()

	alerts := make([]*storage.Alert, 0, 4000)
	for i := 0; i < 4000; i++ {
		a := fixtures.GetAlert()
		a.Deployment.Containers = nil
		a.Id = fmt.Sprintf("%d", i)
		alerts = append(alerts, a)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.NoError(b, indexer.AddAlerts(alerts))
	}
}
