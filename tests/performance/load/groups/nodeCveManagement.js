import { group } from 'k6';
import http from 'k6/http';

export function nodeCveManagement(host, headers, tags) {
    group('node cve management', function () {
        http.batch([
            [
                'POST',
                `${host}/api/graphql?opname=getNodeCVEs`,
                JSON.stringify({
                    operationName: 'getNodeCVEs',
                    variables: {
                        query: '',
                        pagination: {
                            offset: 0,
                            limit: 20,
                            sortOption: {
                                field: 'CVSS',
                                reversed: true,
                                aggregateBy: {
                                    aggregateFunc: 'max',
                                    distinct: false,
                                },
                            },
                        },
                    },
                    query: `query getNodeCVEs($query: String, $pagination: Pagination) {
  nodeCVEs(query: $query, pagination: $pagination) {
    cve
    nodeCountBySeverity {
      critical { total __typename }
      important { total __typename }
      moderate { total __typename }
      low { total __typename }
      __typename
    }
    topCVSS
    affectedNodeCount
    firstDiscoveredInSystem
    distroTuples { summary operatingSystem __typename }
    __typename
  }
}`,
                }),
                { headers, tags },
            ],
            [
                'POST',
                `${host}/api/graphql?opname=getTotalNodeCount`,
                JSON.stringify({
                    operationName: 'getTotalNodeCount',
                    variables: {},
                    query: `query getTotalNodeCount { nodeCount }`,
                }),
                { headers, tags },
            ],
            [
                'POST',
                `${host}/api/graphql?opname=getSnoozedNodeCveCount`,
                JSON.stringify({
                    operationName: 'getSnoozedNodeCveCount',
                    variables: {},
                    query: `query getSnoozedNodeCveCount { nodeCVECount(query: "CVE Snoozed:true") }`,
                }),
                { headers, tags },
            ],
            [
                'POST',
                `${host}/api/graphql?opname=getNodeCVEEntityCounts`,
                JSON.stringify({
                    operationName: 'getNodeCVEEntityCounts',
                    variables: { query: '' },
                    query: `query getNodeCVEEntityCounts($query: String) {
  nodeCount(query: $query)
  nodeCVECount(query: $query)
}`,
                }),
                { headers, tags },
            ],
        ]);
    });
}
