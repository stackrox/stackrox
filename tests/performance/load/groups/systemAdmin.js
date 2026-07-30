import { group } from 'k6';
import http from 'k6/http';

export function systemAdmin(host, headers, tags) {
    group('system administration', function () {
        http.batch([
            ['GET', `${host}/v1/config`, null, { headers, tags }],
            ['GET', `${host}/v1/database/status`, null, { headers, tags }],
            [
                'GET',
                `${host}/v1/administration/events?pagination.offset=0&pagination.limit=10&pagination.sortOption.field=${encodeURIComponent('Last Updated')}&pagination.sortOption.reversed=true`,
                null,
                { headers, tags },
            ],
            [
                'GET',
                `${host}/v1/count/administration/events`,
                null,
                { headers, tags },
            ],
            ['GET', `${host}/v1/clusters`, null, { headers, tags }],
            [
                'GET',
                `${host}/v1/credentialexpiry?component=CENTRAL`,
                null,
                { headers, tags },
            ],
            [
                'GET',
                `${host}/v1/administration/usage/secured-units/current`,
                null,
                { headers, tags },
            ],
            [
                'GET',
                `${host}/v1/administration/usage/secured-units/max?from=${new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString()}&to=${new Date().toISOString()}`,
                null,
                { headers, tags },
            ],
        ]);
    });
}
