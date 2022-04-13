package tests

import (
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/roxctl/common"
)

func getAlertsSummaryByCluster(service v1.AlertServiceClient) func() error {
	return getAlertsSummary(service, v1.GetAlertsCountsRequest_CLUSTER)
}

func getAlertsSummaryByCategory(service v1.AlertServiceClient) func() error {
	return getAlertsSummary(service, v1.GetAlertsCountsRequest_CATEGORY)
}

func getAlertsSummary(service v1.AlertServiceClient, groupBy v1.GetAlertsCountsRequest_RequestGroup) func() error {
	return func() error {
		alertCountsRequest := &v1.GetAlertsCountsRequest{
			Request: &v1.ListAlertsRequest{
				Query: "",
			},
			GroupBy: groupBy,
		}
		_, err := service.GetAlertsCounts(common.Context(), alertCountsRequest)
		return err
	}
}

func getAlertsSummaryTimeseries(service v1.AlertServiceClient) func() error {
	return func() error {
		request := &v1.ListAlertsRequest{
			Query: "",
		}
		_, err := service.GetAlertTimeseries(common.Context(), request)
		return err
	}
}

func getDeploymentsWithProcessInfo(service v1.DeploymentServiceClient) func() error {
	return func() error {
		query := &v1.RawQuery{
			Query: "",
		}
		_, err := service.ListDeploymentsWithProcessInfo(common.Context(), query)
		return err
	}
}

func getSummary(service v1.SummaryServiceClient) func() error {
	return func() error {
		_, err := service.GetSummaryCounts(common.Context(), &v1.Empty{})
		return err
	}
}

func BenchmarkDashboard(b *testing.B) {
	envVars := getEnvVars()

	connection, err := getConnection(envVars.endpoint, envVars.password)
	if err != nil {
		log.Fatal(err)
	}

	alertService := v1.NewAlertServiceClient(connection)
	deploymentService := v1.NewDeploymentServiceClient(connection)
	summaryService := v1.NewSummaryServiceClient(connection)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		wg := concurrency.NewWaitGroup(0)
		asyncWithWaitGroup(getAlertsSummaryByCluster(alertService), &wg)
		asyncWithWaitGroup(getDeploymentsWithProcessInfo(deploymentService), &wg)
		asyncWithWaitGroup(getSummary(summaryService), &wg)
		asyncWithWaitGroup(getAlertsSummaryByCategory(alertService), &wg)
		asyncWithWaitGroup(getAlertsSummaryTimeseries(alertService), &wg)
		<-wg.Done()
	}
}
