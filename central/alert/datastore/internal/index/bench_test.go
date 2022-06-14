package index

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listAlertFixture() *storage.ListAlert {
	return convert.AlertToListAlert(fixtures.GetAlert())
}

func getAlertIndex() Indexer {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	if err != nil {
		panic(err)
	}
	return New(tmpIndex)
}

func benchmarkAddAlertNumThen1(b *testing.B, numAlerts int) {
	indexer := getAlertIndex()
	alert := listAlertFixture()
	addAlerts(b, indexer, alert, numAlerts)
	alert.Id = fmt.Sprintf("%d", numAlerts+1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.NoError(b, indexer.AddListAlert(alert))
	}
}

func addAlerts(b *testing.B, indexer Indexer, alert *storage.ListAlert, numAlerts int) {
	for i := 0; i < numAlerts; i++ {
		alert.Id = fmt.Sprintf("%d", i)
		require.NoError(b, indexer.AddListAlert(alert))
	}
}

func benchmarkAddAlert(b *testing.B, numAlerts int) {
	indexer := getAlertIndex()
	alert := listAlertFixture()
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

func BenchmarkIndex(b *testing.B) {
	indexer := getAlertIndex()

	totalAlerts := 20000
	alerts := make([]*storage.ListAlert, 0, totalAlerts)
	for i := 0; i < totalAlerts; i++ {
		alert := listAlertFixture()
		alert.Id = fmt.Sprintf("%d", i)
		alerts = append(alerts, alert)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		assert.NoError(b, indexer.AddListAlerts(alerts))
	}
}
