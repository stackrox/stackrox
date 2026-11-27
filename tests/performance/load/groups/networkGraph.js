import { group } from 'k6';
import http from 'k6/http';

export function networkGraph(host, headers, tags) {
    group('network graph', function () {
        const clustersResp = http.get(
            `${host}/v1/sac/clusters?permissions=Deployment`,
            { headers, tags }
        );

        let clusterId = '';
        try {
            const clusters = JSON.parse(clustersResp.body);
            if (clusters.clusters && clusters.clusters.length > 0) {
                clusterId = clusters.clusters[0].id;
            }
        } catch (_) {}

        if (clusterId) {
            const since = new Date(Date.now() - 60 * 60 * 1000).toISOString();
            http.batch([
                ['GET', `${host}/v1/networkpolicies?cluster_id=${clusterId}`, null, { headers, tags }],
                ['GET', `${host}/v1/networkgraph/cluster/${clusterId}?since=${since}`, null, { headers, tags }],
                ['GET', `${host}/v1/networkgraph/config`, null, { headers, tags }],
            ]);
        }
    });
}
