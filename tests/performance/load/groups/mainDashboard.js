import { group } from 'k6';
import http from 'k6/http';
import { URLSearchParams } from 'https://jslib.k6.io/url/1.0.0/index.js';

function getDeploymentAtMostRiskSearchParams() {
    const params = new URLSearchParams([
        ['pagination', [
            ['offset', 0],
            ['limit', 6],
            ['sortOption', [
                ['field', 'Deployment Risk Priority'],
                ['reversed', 'false'],
            ]]
        ]],
    ]);
    return params.toString();
}

function getAlertSummaryCountSearchParams() {
    const params = new URLSearchParams([
        ['request', [
            ['query', '']
        ]],
        ['group_by', 'CATEGORY'],
    ]);
    return params.toString();

}

export function mainDashboard(host, headers, tags) {
    group('main dashboard', function () {
        http.post(
            `${host}/api/graphql?opname=summary_counts`,
            JSON.stringify({
                "operationName":"summary_counts",
                "variables":{},
                "query":`query summary_counts {
  clusterCount
  nodeCount
  violationCount
  deploymentCount
  imageCount
  secretCount
}`
            }),
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=getAllNamespacesByCluster`,
            JSON.stringify({
                "operationName":"getAllNamespacesByCluster",
                "variables":{
                    "query":""
                },
                "query":`query getAllNamespacesByCluster($query: String) {
  clusters(query: $query) {
    id
    name
    namespaces {
      metadata {
        id
        name
        __typename
      }
      __typename
    }
    __typename
  }
}`
            }),
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=alertCountsBySeverity`,
            JSON.stringify({
                "operationName":"alertCountsBySeverity",
                "variables":{
                    "lowQuery":"Severity:LOW_SEVERITY",
                    "medQuery":"Severity:MEDIUM_SEVERITY",
                    "highQuery":"Severity:HIGH_SEVERITY",
                    "critQuery":"Severity:CRITICAL_SEVERITY"
                },
                "query":`query alertCountsBySeverity($lowQuery: String, $medQuery: String, $highQuery: String, $critQuery: String) {
  LOW_SEVERITY: violationCount(query: $lowQuery)
  MEDIUM_SEVERITY: violationCount(query: $medQuery)
  HIGH_SEVERITY: violationCount(query: $highQuery)
  CRITICAL_SEVERITY: violationCount(query: $critQuery)
}`
            }),
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=mostRecentAlerts`,
            JSON.stringify({
                "operationName":"mostRecentAlerts",
                "variables":{
                    "query":"Severity:CRITICAL_SEVERITY"
                },
                "query":`query mostRecentAlerts($query: String) {
  alerts: violations(
    query: $query
    pagination: {limit: 3, sortOption: {field: "Violation Time", reversed: true}}
  ) {
    id
    time
    deployment {
      name
      __typename
    }
    resource {
      resourceType
      name
      __typename
    }
    policy {
      name
      severity
      __typename
    }
    __typename
  }
}`
            }),
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=getImagesAtMostRisk`,
            JSON.stringify({
                "operationName":"getImagesAtMostRisk",
                "variables":{
                    "query":""
                },
                "query":`query getImagesAtMostRisk($query: String) {
  images(
    query: $query
    pagination: {limit: 6, sortOption: {field: "Image Risk Priority", reversed: false}}
  ) {
    id
    name {
      remote
      fullName
      __typename
    }
    priority
    imageVulnerabilityCounter {
      important {
        total
        fixable
        __typename
      }
      critical {
        total
        fixable
        __typename
      }
      __typename
    }
    __typename
  }
}`
            }),
            { headers, tags }
        );

        http.get(
            `${host}/v1/deploymentswithprocessinfo?${getDeploymentAtMostRiskSearchParams()}`,
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=agingImagesQuery`,
            JSON.stringify({
                "operationName":"agingImagesQuery",
                "variables":{
                    "query0":"Image Created Time:30d-90d",
                    "query1":"Image Created Time:90d-180d",
                    "query2":"Image Created Time:180d-365d",
                    "query3":"Image Created Time:>365d"
                },
                "query":`query agingImagesQuery($query0: String, $query1: String, $query2: String, $query3: String) {
  timeRange0: imageCount(query: $query0)
  timeRange1: imageCount(query: $query1)
  timeRange2: imageCount(query: $query2)
  timeRange3: imageCount(query: $query3)
}`
            }),
            { headers, tags }
        );

        http.get(
            `${host}/v1/alerts/summary/counts?${getAlertSummaryCountSearchParams()}`,
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=getAggregatedResults`,
            JSON.stringify({
                "operationName":"getAggregatedResults",
                "variables":{
                    "groupBy":["STANDARD"],
                    "where":"Cluster:*"
                },
                "query":`query getAggregatedResults($groupBy: [ComplianceAggregation_Scope!], $where: String) {
  controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {
    results {
      aggregationKeys {
        id
        scope
        __typename
      }
      numFailing
      numPassing
      numSkipped
      unit
      __typename
    }
    __typename
  }
  complianceStandards: complianceStandards {
    id
    name
    __typename
  }
}`
            }),
            { headers, tags }
        );
    });
}
