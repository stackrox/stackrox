import { getURLLinkToDeployment } from './networkGraphURLUtils';

describe('networkGraphURLUtils', () => {
    describe('getURLLinkToDeployment', () => {
        it('should get the URL to a specific deployment in the network graph', () => {
            const cluster = 'remote';
            const namespace = 'stackrox';
            const deploymentId = '8cbfde79-3450-45bb-a5c9-4185b9d1d0f1';
            const url = getURLLinkToDeployment({ cluster, namespace, deploymentId });
            expect(url).toEqual(
                '/main/network-graph/deployment/8cbfde79-3450-45bb-a5c9-4185b9d1d0f1?s[Cluster]=remote&s[Namespace]=stackrox'
            );
        });
    });
});
