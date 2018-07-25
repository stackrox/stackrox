package index

import (
	"fmt"
	"testing"

	"bitbucket.org/stack-rox/apollo/central/globalindex"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/fixtures"
	"bitbucket.org/stack-rox/apollo/pkg/search"
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
	addAlerts(indexer, alert, numAlerts)
	alert.Id = fmt.Sprintf("%d", numAlerts+1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexer.AddAlert(alert)
	}
}

func addAlerts(indexer Indexer, alert *v1.Alert, numAlerts int) {
	for i := 0; i < numAlerts; i++ {
		alert.Id = fmt.Sprintf("%d", i)
		indexer.AddAlert(alert)
	}
}

func benchmarkAddAlert(b *testing.B, numAlerts int) {
	indexer := getAlertIndex()
	alert := fixtures.GetAlert()
	for i := 0; i < b.N; i++ {
		addAlerts(indexer, alert, numAlerts)
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
		indexer.SearchAlerts(qb.ToParsedSearchRequest())
	}
}
