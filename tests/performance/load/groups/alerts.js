import {Client, StatusOK} from 'k6/net/grpc';
import {check, group} from 'k6';

const GRPC_PORT = ":443";

const client = new Client();

client.load(['../../../proto', '../../../third_party/googleapis'], 'api/v1/alert_service.proto');

export function alertsGrpc(host, headers, tags) {

    host = host.replace('https://', '')
    client.connect(host + GRPC_PORT);

    const params = {
        metadata: headers,
    };

    [0, 10, 100, 1000].forEach(limit => {
        group('list alerts grpc', function () {
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

            check(response, {'status is OK': (r) => r && r.status === StatusOK && r.message.alerts.length > 0});
        })
    });
    client.close();
}