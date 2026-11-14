import { group } from 'k6';
import http from 'k6/http';

export function integrations(host, headers, tags) {
    group('integrations', function () {
        http.batch([
            ['GET', `${host}/v1/imageintegrations`, null, { headers, tags }],
            ['GET', `${host}/v1/notifiers`, null, { headers, tags }],
            ['GET', `${host}/v1/externalbackups`, null, { headers, tags }],
            ['GET', `${host}/v1/signatureintegrations`, null, { headers, tags }],
            ['GET', `${host}/v1/cloud-sources`, null, { headers, tags }],
            ['GET', `${host}/v1/delegatedregistryconfig`, null, { headers, tags }],
            ['GET', `${host}/v1/integrationhealth/imageintegrations`, null, { headers, tags }],
            ['GET', `${host}/v1/integrationhealth/notifiers`, null, { headers, tags }],
            ['GET', `${host}/v1/integrationhealth/externalbackups`, null, { headers, tags }],
        ]);
    });
}
