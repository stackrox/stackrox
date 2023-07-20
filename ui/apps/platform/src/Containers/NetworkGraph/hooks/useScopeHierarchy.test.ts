import { getScopeHierarchyFromSearch } from './useScopeHierarchy';

describe('useScopeHierarchy', () => {
    describe('getScopeHierarchyFromSearch', () => {
        it('should return the correct cluster in the result hierarchy', () => {
            const knownClusters = [
                { id: '1', name: 'cluster1' },
                { id: '2', name: 'cluster2' },
            ];

            expect(getScopeHierarchyFromSearch({}, knownClusters)).toBeNull();
            expect(getScopeHierarchyFromSearch({ Cluster: undefined }, knownClusters)).toBeNull();
            expect(getScopeHierarchyFromSearch({ Cluster: 'cluster3' }, knownClusters)).toBeNull();
            expect(
                getScopeHierarchyFromSearch({ Cluster: ['cluster1', 'cluster2'] }, knownClusters)
            ).toBeNull();

            // The only supported case is a non-array Cluster value that matches a known cluster
            expect(
                getScopeHierarchyFromSearch({ Cluster: 'cluster2' }, knownClusters)?.cluster.id
            ).toBe('2');
        });

        it('should correctly retain namespaces, deployments, and remaining query values', () => {
            const knownClusters = [
                { id: '1', name: 'cluster1' },
                { id: '2', name: 'cluster2' },
            ];

            const searchFilter = {
                Cluster: 'cluster1',
                Namespace: 'namespace1',
                Deployment: ['deployment1', 'deployment2'],
                CVE: 'CVE-2020-1234',
                Other: 'other',
            };

            expect(getScopeHierarchyFromSearch(searchFilter, knownClusters)).toEqual({
                cluster: { id: '1', name: 'cluster1' },
                namespaces: ['namespace1'],
                deployments: ['deployment1', 'deployment2'],
                remainingQuery: {
                    CVE: 'CVE-2020-1234',
                    Other: 'other',
                },
            });
        });
    });
});
