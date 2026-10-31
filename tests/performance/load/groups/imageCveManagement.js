import { group } from 'k6';
import http from 'k6/http';

export function imageCveManagement(host, headers, tags) {
    group('image cve management', function () {
        const query =
            'Platform Component:false+Severity:CRITICAL_VULNERABILITY_SEVERITY,IMPORTANT_VULNERABILITY_SEVERITY+Fixable:true+Vulnerability State:OBSERVED';

        http.batch([
            [
                'POST',
                `${host}/api/graphql?opname=getImageCVEList`,
                JSON.stringify({
                    operationName: 'getImageCVEList',
                    variables: {
                        query,
                        pagination: {
                            offset: 0,
                            limit: 20,
                            sortOptions: [
                                {
                                    field: 'Critical Severity Count',
                                    reversed: true,
                                },
                            ],
                        },
                    },
                    query: `query getImageCVEList($query: String, $pagination: Pagination) {
  imageCVEs(query: $query, pagination: $pagination) {
    cve
    affectedImageCountBySeverity {
      critical { total __typename }
      important { total __typename }
      moderate { total __typename }
      low { total __typename }
      __typename
    }
    topCVSS
    affectedImageCount
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
                `${host}/api/graphql?opname=getUnfilteredImageCount`,
                JSON.stringify({
                    operationName: 'getUnfilteredImageCount',
                    variables: {},
                    query: `query getUnfilteredImageCount {
  imageCount
}`,
                }),
                { headers, tags },
            ],
            [
                'POST',
                `${host}/api/graphql?opname=getEntityTypeCounts`,
                JSON.stringify({
                    operationName: 'getEntityTypeCounts',
                    variables: { query },
                    query: `query getEntityTypeCounts($query: String) {
  imageCount(query: $query)
  deploymentCount(query: $query)
  imageCVECount(query: $query)
}`,
                }),
                { headers, tags },
            ],
            ['GET', `${host}/v1/watchedimages`, null, { headers, tags }],
        ]);
    });
}
