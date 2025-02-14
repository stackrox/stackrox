import {Client, StatusOK} from 'k6/net/grpc';
import {check, group} from 'k6';

const GRPC_PORT = ":443";

const client = new Client();

client.load(['proto/stackrox', 'proto/googleapis'], 'api/v1/alert_service.proto');

export function alertsGrpc(host, headers, tags) {

    host = host.replace('https://', '')
    host = host.includes(':') ? host : host + GRPC_PORT;
    client.connect(host);

    group('list alerts grpc', function () {
    [0, 10, 100, 1000].forEach(limit => {
        tags.limit = limit;
        const params = {
            metadata: headers, tags: tags,
        };
        const response = client.invoke(
            'v1.AlertService/ListAlerts',
            {
                pagination: {
                    limit: limit,
                    offset: 0,
                    sortOption: {
                        field: 'Violation Time'
                    }
                }
            },
            params,
        );
        tags.fetched = response?.message?.alerts?.length ?? 0;
        check(response, {'status is OK': (r) => r && r.status === StatusOK && tags.fetched > 0}, tags);
    });
    });
    client.close();
}
