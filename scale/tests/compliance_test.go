package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/stackrox/central/compliance/standards/metadata"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/roxctl/common"
)

var (
	aggregateQueries = []aggregateQuery{
		{
			[]string{"STANDARD", "CLUSTER"},
			"CHECK",
		},
		{
			[]string{"STANDARD", "CLUSTER"},
			"CHECK",
		},
		{
			[]string{"STANDARD", "NAMESPACE"},
			"CHECK",
		},
		{
			[]string{"STANDARD", "NODE"},
			"CHECK",
		},
	}

	standardNames = func() map[string]string {
		m := make(map[string]string)
		for _, standard := range metadata.AllStandards {
			m[standard.ID] = standard.Name
		}
		return m
	}()

	maxWaitTime = 20 * time.Minute
	maxRetries  = 1
)

type aggregateQuery struct {
	groupBy []string
	unit    string
}

func sendGraphql(endpoint, password string, query []byte, client *http.Client) []byte {
	endpoint = fmt.Sprintf("https://%s/api/graphql", endpoint)
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(query))
	if err != nil {
		panic(err)
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq = httpReq.WithContext(common.Context())
	httpReq.SetBasicAuth("admin", password)
	resp, err := client.Do(httpReq)
	if err != nil {
		panic(err)
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return respBytes
}

func allFinished(runResponses []complianceRunsListItem) bool {
	for _, response := range runResponses {
		if response.State != "FINISHED" {
			return false
		}
	}
	return true
}

func getRunIDs(runResponses []complianceRunsListItem) []string {
	ids := make([]string, len(runResponses))
	for i, response := range runResponses {
		ids[i] = response.ID
	}
	return ids
}

const (
	triggerScanQuery = "mutation triggerScan($clusterId: ID!, $standardId: ID!) {\n  complianceTriggerRuns(clusterId: $clusterId, standardId: $standardId) {\n    id\n    standardId\n    clusterId\n    state\n    errorMessage\n    __typename\n  }\n}\n"
)

func getTriggerScan(clusterID, standardID string) []byte {
	return marshallQuery(
		triggerScanQuery,
		"triggerScan",
		map[string]interface{}{
			"clusterId":  clusterID,
			"standardId": standardID,
		},
	)
}

func getTriggerScanResult(resp []byte) []complianceRunsListItem {
	var unmarshalledResp triggerScanResponse
	if err := json.Unmarshal(resp, &unmarshalledResp); err != nil {
		log.Error(string(resp))
		panic(err)
	}
	return unmarshalledResp.Data.ComplianceTriggerRuns
}

func triggerAndWaitForCompliance(envVars *testEnvVars, client *http.Client) ([]complianceRunsListItem, error) {
	query := getTriggerScan("*", "*")

	log.Info("triggering compliance run")
	resp := sendGraphql(envVars.endpoint, envVars.password, query, client)
	respList := getTriggerScanResult(resp)
	runIDs := getRunIDs(respList)
	startTime := time.Now()
	for !allFinished(respList) {
		time.Sleep(time.Second * 5)
		if time.Since(startTime) > maxWaitTime {
			return nil, fmt.Errorf("max wait time for compliance run exceeded, last set of responses was %s", respList)
		}
		fmt.Print(".")
		query := getRunStatuses(runIDs)
		resp := sendGraphql(envVars.endpoint, envVars.password, query, client)
		respList = getRunStatusesResult(resp)
	}
	return respList, nil
}

func getSendFunc(endpoint, password string, query []byte, client *http.Client) func() error {
	return func() error {
		sendGraphql(endpoint, password, query, client)
		return nil
	}
}

func makeComplianceQueries(runs []complianceRunsListItem) [][]byte {
	var graphqlQueries [][]byte
	graphqlQueries = append(graphqlQueries, getSummaryCounts())
	graphqlQueries = append(graphqlQueries, getClustersCount())
	graphqlQueries = append(graphqlQueries, getNamespacesCount())
	graphqlQueries = append(graphqlQueries, getNodesCount())
	graphqlQueries = append(graphqlQueries, getDeploymentsCount())
	for _, query := range aggregateQueries {
		graphqlQueries = append(graphqlQueries, getAggregatedResults(query.groupBy, query.unit))
	}
	for _, run := range runs {
		standardName, ok := standardNames[run.StandardID]
		if !ok {
			continue
		}
		standardName = fmt.Sprintf("Standard:%s", standardName)
		graphqlQueries = append(graphqlQueries, getComplianceStandards(standardName))
	}
	return graphqlQueries
}

func loadComplianceResults(envVars *testEnvVars, client *http.Client, graphqlQueries [][]byte) {
	log.Info("begin loading compliance results")
	wg := concurrency.NewWaitGroup(0)
	for _, query := range graphqlQueries {
		syncFunc := getSendFunc(envVars.endpoint, envVars.password, query, client)
		asyncWithWaitGroup(syncFunc, &wg)
	}
	<-wg.Done()
	log.Info("finished loading compliance results")
}

func BenchmarkCompliance(b *testing.B) {
	envVars := getEnvVars()
	log.Infof("benchmarking compliance against %s", envVars.endpoint)
	client := getHTTPClient()

	// Run compliance
	retries := 0
	var complianceRuns []complianceRunsListItem
	var err error
	for ; complianceRuns == nil && retries <= maxRetries; retries++ {
		complianceRuns, err = triggerAndWaitForCompliance(envVars, client)
		if err != nil {
			log.Info(err)
		}
	}
	if complianceRuns == nil {
		panic("Unable to get a complete compliance run")
	}
	log.Info("completed compliance run, beginning benchmark")
	// Build the queries we are going to run
	complianceQueries := makeComplianceQueries(complianceRuns)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Run all queries asynchronously and wait for each to finish
		loadComplianceResults(envVars, client, complianceQueries)
	}
}
