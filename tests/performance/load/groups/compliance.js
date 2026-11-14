import { group } from 'k6';
import http from 'k6/http';

export function compliance(host, headers, tags) {
    group('compliance', function () {
        http.batch([
            ['GET', `${host}/v1/compliance/standards`, null, { headers, tags }],
            [
                'POST',
                `${host}/api/graphql?opname=runStatuses`,
                JSON.stringify({
                    operationName: 'runStatuses',
                    variables: {},
                    query: `query runStatuses {
  complianceRunStatuses {
    runs {
      id
      standardId
      clusterId
      state
      errorMessage
      __typename
    }
    __typename
  }
}`,
                }),
                { headers, tags },
            ],
        ]);

        http.get(
            `${host}/v2/compliance/scan/configurations?query.pagination.offset=0&query.pagination.limit=10&query.pagination.sortOption.field=${encodeURIComponent('Compliance Scan Config Name')}&query.pagination.sortOption.reversed=false`,
            { headers, tags }
        );
    });
}
