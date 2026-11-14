import { group } from 'k6';
import http from 'k6/http';

export function accessControl(host, headers, tags) {
    group('access control', function () {
        http.batch([
            ['GET', `${host}/v1/roles`, null, { headers, tags }],
            ['GET', `${host}/v1/permissionsets`, null, { headers, tags }],
            ['GET', `${host}/v1/simpleaccessscopes`, null, { headers, tags }],
            ['GET', `${host}/v1/authProviders`, null, { headers, tags }],
            ['GET', `${host}/v1/groups`, null, { headers, tags }],
            ['GET', `${host}/v1/auth/m2m`, null, { headers, tags }],
        ]);
    });
}
