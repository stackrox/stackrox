import { group } from 'k6';
import http from 'k6/http';

export function loginInit(host, headers, tags) {
    group('login init', function () {
        http.batch([
            ['GET', `${host}/v1/availableAuthProviders`, null, { headers, tags }],
            ['GET', `${host}/v1/central-capabilities`, null, { headers, tags }],
            ['GET', `${host}/v1/login/authproviders`, null, { headers, tags }],
            ['GET', `${host}/v1/mypermissions`, null, { headers, tags }],
            ['GET', `${host}/v1/metadata`, null, { headers, tags }],
            ['GET', `${host}/v1/config/public`, null, { headers, tags }],
            ['GET', `${host}/v1/featureflags`, null, { headers, tags }],
            ['GET', `${host}/v1/auth/status`, null, { headers, tags }],
            ['GET', `${host}/v1/telemetry/config`, null, { headers, tags }],
            ['GET', `${host}/v1/database/status`, null, { headers, tags }],
        ]);

        http.post(
            `${host}/api/graphql?opname=healths`,
            JSON.stringify({
                operationName: 'healths',
                variables: {},
                query: `query healths {
  clusters: clusterHealth {
    id
    healthStatus {
      overallHealthStatus
      __typename
    }
    __typename
  }
}`,
            }),
            { headers, tags }
        );
    });
}
