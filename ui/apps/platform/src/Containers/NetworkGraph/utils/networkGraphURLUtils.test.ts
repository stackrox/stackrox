import { getPropertiesForAnalytics } from './networkGraphURLUtils';

describe('networkGraphURLUtils', () => {
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
