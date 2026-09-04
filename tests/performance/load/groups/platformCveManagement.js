import { group } from 'k6';
import http from 'k6/http';

export function platformCveManagement(host, headers, tags) {
    group('platform cve management', function () {
        http.batch([
            [
                'POST',
                `${host}/api/graphql?opname=getPlatformCves`,
                JSON.stringify({
                    operationName: 'getPlatformCves',
                    variables: {
                        query: '',
                        pagination: {
                            offset: 0,
                            limit: 20,
                            sortOption: {
                                field: 'CVSS',
                                reversed: true,
                            },
                        },
                    },
                    query: `query getPlatformCves($query: String, $pagination: Pagination) {
  platformCVEs(query: $query, pagination: $pagination) {
    cve
    clusterCountByType { generic kubernetes openshift __typename }
    topCVSS
    affectedClusterCount
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
                `${host}/api/graphql?opname=getPlatformCVEEntityCounts`,
                JSON.stringify({
                    operationName: 'getPlatformCVEEntityCounts',
                    variables: { query: '' },
                    query: `query getPlatformCVEEntityCounts($query: String) {
  clusterCount(query: $query)
  platformCVECount(query: $query)
}`,
                }),
                { headers, tags },
            ],
            [
                'POST',
                `${host}/api/graphql?opname=getTotalClusterCount`,
                JSON.stringify({
                    operationName: 'getTotalClusterCount',
                    variables: {},
                    query: `query getTotalClusterCount { clusterCount }`,
                }),
                { headers, tags },
            ],
            [
                'POST',
                `${host}/api/graphql?opname=getSnoozedPlatformCveCount`,
                JSON.stringify({
                    operationName: 'getSnoozedPlatformCveCount',
                    variables: {},
                    query: `query getSnoozedPlatformCveCount { platformCVECount(query: "CVE Snoozed:true") }`,
                }),
                { headers, tags },
            ],
        ]);
    });
}
