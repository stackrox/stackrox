import { getURLLinkToDeployment, getPropertiesForAnalytics } from './networkGraphURLUtils';

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

    describe('getPropertiesForAnalytics', () => {
        it('should properties with just a cluster', () => {
            const searchFilter = {
                Cluster: 'staging-secured-cluster',
                Namespace: undefined,
                Deployment: undefined,
            };

            const properties = getPropertiesForAnalytics(searchFilter);

            expect(properties).toEqual({
                cluster: 1,
                namespaces: 0,
                deployments: 0,
            });
        });

        it('should return properties with just cluster and namespaces', () => {
            const searchFilter = {
                Cluster: 'staging-secured-cluster',
                Namespace: ['default', 'stackrox'],
                Deployment: undefined,
            };

            const properties = getPropertiesForAnalytics(searchFilter);

            expect(properties).toEqual({
                cluster: 1,
                namespaces: 2,
                deployments: 0,
            });
        });

        it('should return properties with cluster, namespaces, and deployments', () => {
            const searchFilter = {
                Cluster: 'staging-secured-cluster',
                Namespace: ['default', 'stackrox', 'payments'],
                Deployment: ['admission-control', 'sensor', 'collector'],
            };

            const properties = getPropertiesForAnalytics(searchFilter);

            expect(properties).toEqual({
                cluster: 1,
                namespaces: 3,
                deployments: 3,
            });
        });

        it('should properties when it does not know the cluster', () => {
            const searchFilter = {
                Cluster: undefined,
                Namespace: undefined,
                Deployment: undefined,
            };

            const properties = getPropertiesForAnalytics(searchFilter);

            expect(properties).toEqual({
                cluster: 0,
                namespaces: 0,
                deployments: 0,
            });
        });
    });
});
