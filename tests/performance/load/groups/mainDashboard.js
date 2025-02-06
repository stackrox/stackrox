import { group } from 'k6';
import http from 'k6/http';

export function mainDashboard(host, headers, tags) {
    group('main dashboard', function () {
        http.post(
            `${host}/api/graphql?opname=summary_counts`,
            '{"operationName":"summary_counts","variables":{},"query":"query summary_counts {\\n  clusterCount\\n  nodeCount\\n  violationCount\\n  deploymentCount\\n  imageCount\\n  secretCount\\n}"}',
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=getAllNamespacesByCluster`,
            '{"operationName":"getAllNamespacesByCluster","variables":{"query":""},"query":"query getAllNamespacesByCluster($query: String) {\\n  clusters(query: $query) {\\n    id\\n    name\\n    namespaces {\\n      metadata {\\n        id\\n        name\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}"}',
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=alertCountsBySeverity`,
            '{"operationName":"alertCountsBySeverity","variables":{"lowQuery":"Severity:LOW_SEVERITY","medQuery":"Severity:MEDIUM_SEVERITY","highQuery":"Severity:HIGH_SEVERITY","critQuery":"Severity:CRITICAL_SEVERITY"},"query":"query alertCountsBySeverity($lowQuery: String, $medQuery: String, $highQuery: String, $critQuery: String) {\\n  LOW_SEVERITY: violationCount(query: $lowQuery)\\n  MEDIUM_SEVERITY: violationCount(query: $medQuery)\\n  HIGH_SEVERITY: violationCount(query: $highQuery)\\n  CRITICAL_SEVERITY: violationCount(query: $critQuery)\\n}"}',
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=mostRecentAlerts`,
            '{"operationName":"mostRecentAlerts","variables":{"query":"Severity:CRITICAL_SEVERITY"},"query":"query mostRecentAlerts($query: String) {\\n  alerts: violations(\\n    query: $query\\n    pagination: {limit: 3, sortOption: {field: \\"Violation Time\\", reversed: true}}\\n  ) {\\n    id\\n    time\\n    deployment {\\n      name\\n      __typename\\n    }\\n    resource {\\n      resourceType\\n      name\\n      __typename\\n    }\\n    policy {\\n      name\\n      severity\\n      __typename\\n    }\\n    __typename\\n  }\\n}"}',
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=getImagesAtMostRisk`,
            '{"operationName":"getImagesAtMostRisk","variables":{"query":""},"query":"query getImagesAtMostRisk($query: String) {\\n  images(\\n    query: $query\\n    pagination: {limit: 6, sortOption: {field: \\"Image Risk Priority\\", reversed: false}}\\n  ) {\\n    id\\n    name {\\n      remote\\n      fullName\\n      __typename\\n    }\\n    priority\\n    imageVulnerabilityCounter {\\n      important {\\n        total\\n        fixable\\n        __typename\\n      }\\n      critical {\\n        total\\n        fixable\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}"}',
            { headers, tags }
        );

        http.get(
            `${host}/v1/deploymentswithprocessinfo?pagination.offset=0&pagination.limit=6&pagination.sortOption.field=Deployment%20Risk%20Priority&pagination.sortOption.reversed=false`,
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=agingImagesQuery`,
            '{"operationName":"agingImagesQuery","variables":{"query0":"Image Created Time:30d-90d","query1":"Image Created Time:90d-180d","query2":"Image Created Time:180d-365d","query3":"Image Created Time:>365d"},"query":"query agingImagesQuery($query0: String, $query1: String, $query2: String, $query3: String) {\\n  timeRange0: imageCount(query: $query0)\\n  timeRange1: imageCount(query: $query1)\\n  timeRange2: imageCount(query: $query2)\\n  timeRange3: imageCount(query: $query3)\\n}"}',
            { headers, tags }
        );

        http.get(
            `${host}/v1/alerts/summary/counts?request.query=&group_by=CATEGORY`,
            { headers, tags }
        );

        http.post(
            `${host}/api/graphql?opname=getAggregatedResults`,
            '{"operationName":"getAggregatedResults","variables":{"groupBy":["STANDARD"],"where":"Cluster:*"},"query":"query getAggregatedResults($groupBy: [ComplianceAggregation_Scope!], $where: String) {\\n  controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {\\n    results {\\n      aggregationKeys {\\n        id\\n        scope\\n        __typename\\n      }\\n      numFailing\\n      numPassing\\n      numSkipped\\n      unit\\n      __typename\\n    }\\n    __typename\\n  }\\n  complianceStandards: complianceStandards {\\n    id\\n    name\\n    __typename\\n  }\\n}"}',
            { headers, tags }
        );
    });
}
