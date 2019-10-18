package tests

import "encoding/json"

type params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

type triggerScanResponse struct {
	Data triggerRunsData
}

type triggerRunsData struct {
	ComplianceTriggerRuns []complianceRunsListItem
}

type runStatusesResponse struct {
	Data runStatusesData
}

type runStatusesData struct {
	ComplianceRunStatuses runStatusesNestedItem
}

type runStatusesNestedItem struct {
	Runs []complianceRunsListItem
}

type complianceRunsListItem struct {
	ID           string
	StandardID   string
	ClusterID    string
	State        string
	ErrorMessage string
}

const (
	getRunStatusesQuery         = "query runStatuses($ids: [ID!]!) {\n  complianceRunStatuses(ids: $ids) {\n    invalidRunIds\n    runs {\n      id\n      standardId\n      clusterId\n      state\n      errorMessage\n      __typename\n    }\n    __typename\n  }\n}\n"
	getSummaryCountsQuery       = "query summary_counts {\n  clusterCount\n  nodeCount\n  violationCount\n  deploymentCount\n  imageCount\n  secretCount\n}\n"
	getClustersCountQuery       = "query clustersCount {\n  results: complianceClusterCount\n}\n"
	getNamespacesCountQuery     = "query namespacesCount {\n  results: complianceNamespaceCount\n}\n"
	getNodesCountQuery          = "query nodesCount {\n  results: complianceNodeCount\n}\n"
	getDeploymentsCountQuery    = "query deploymentsCount {\n  results: complianceDeploymentCount\n}\n"
	getAggregatedResultsQuery   = "query getAggregatedResults($groupBy: [ComplianceAggregation_Scope!], $unit: ComplianceAggregation_Scope!, $where: String) {\n  results: aggregatedResults(groupBy: $groupBy, unit: $unit, where: $where) {\n    results {\n      aggregationKeys {\n        id\n        scope\n        __typename\n      }\n      numFailing\n      numPassing\n      unit\n      __typename\n    }\n    __typename\n  }\n  controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {\n    results {\n      __typename\n      aggregationKeys {\n        __typename\n        id\n        scope\n      }\n      numFailing\n      numPassing\n      unit\n    }\n    __typename\n  }\n  complianceStandards: complianceStandards {\n    id\n    name\n    __typename\n  }\n}\n"
	getComplianceStandardsQuery = "query complianceStandards($groupBy: [ComplianceAggregation_Scope!], $where: String) {\n  complianceStandards {\n    id\n    name\n    controls {\n      standardId\n      groupId\n      id\n      name\n      description\n      __typename\n    }\n    groups {\n      standardId\n      id\n      name\n      description\n      __typename\n    }\n    __typename\n  }\n  results: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {\n    results {\n      aggregationKeys {\n        id\n        scope\n        __typename\n      }\n      numFailing\n      numPassing\n      unit\n      __typename\n    }\n    __typename\n  }\n  checks: aggregatedResults(groupBy: $groupBy, unit: CHECK, where: $where) {\n    results {\n      aggregationKeys {\n        id\n        scope\n        __typename\n      }\n      numFailing\n      numPassing\n      unit\n      __typename\n    }\n    __typename\n  }\n}\n"
)

func getTriggerScanResult(resp []byte) []complianceRunsListItem {
	var unmarshalledResp triggerScanResponse
	if err := json.Unmarshal(resp, &unmarshalledResp); err != nil {
		log.Error(string(resp))
		panic(err)
	}
	return unmarshalledResp.Data.ComplianceTriggerRuns
}

func getRunStatuses(ids []string) []byte {
	return marshallQuery(
		getRunStatusesQuery,
		"runStatuses",
		map[string]interface{}{
			"ids": ids,
		},
	)
}

func getRunStatusesResult(resp []byte) []complianceRunsListItem {
	var unmarshalleResp runStatusesResponse
	if err := json.Unmarshal(resp, &unmarshalleResp); err != nil {
		panic(err)
	}
	return unmarshalleResp.Data.ComplianceRunStatuses.Runs
}

func getSummaryCounts() []byte {
	return marshallQuery(getSummaryCountsQuery, "summary_counts", nil)
}

func getClustersCount() []byte {
	return marshallQuery(getClustersCountQuery, "clustersCount", nil)
}

func getNamespacesCount() []byte {
	return marshallQuery(getNamespacesCountQuery, "namespacesCount", nil)
}

func getNodesCount() []byte {
	return marshallQuery(getNodesCountQuery, "nodesCount", nil)
}

func getDeploymentsCount() []byte {
	return marshallQuery(getDeploymentsCountQuery, "deploymentsCount", nil)
}

func getAggregatedResults(groupBy []string, unit string) []byte {
	return marshallQuery(
		getAggregatedResultsQuery,
		"getAggregatedResults",
		map[string]interface{}{
			"groupBy": groupBy,
			"unit":    unit,
		},
	)
}

func getComplianceStandards(where string) []byte {
	return marshallQuery(
		getComplianceStandardsQuery,
		"complianceStandards",
		map[string]interface{}{
			"groupBy": []string{"STANDARD", "CATEGORY", "CONTROL"},
			"where":   where,
		},
	)
}

func marshallQuery(query, opName string, variables map[string]interface{}) []byte {
	queryParams := params{
		Query:         query,
		OperationName: opName,
		Variables:     variables,
	}
	bytes, err := json.Marshal(queryParams)
	if err != nil {
		panic(err)
	}
	return bytes
}
