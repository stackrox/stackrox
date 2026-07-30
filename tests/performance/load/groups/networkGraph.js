import { group } from 'k6';
import http from 'k6/http';

export function networkGraph(host, headers, tags) {
    group('network graph', function () {
        http.get(`${host}/v1/sac/clusters?permissions=Deployment`, {
            headers,
            tags,
        });

        http.get(`${host}/v1/networkpolicies`, { headers, tags });

        const since = new Date(Date.now() - 60 * 60 * 1000).toISOString();
        http.get(
            `${host}/v1/networkgraph/cluster?clusterId=&since=${since}`,
            { headers, tags }
        );

        http.get(`${host}/v1/networkgraph/config`, { headers, tags });
    });
}
