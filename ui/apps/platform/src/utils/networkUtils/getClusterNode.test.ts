import { getClusterNode } from './getClusterNode';

describe('getClusterNode', () => {
    it('should return a cluster node in the correct data structure', () => {
        const clusterNode = getClusterNode('remote');

        expect(clusterNode).toEqual({
            classes: 'cluster',
            data: {
                id: 'remote',
                name: 'remote',
                active: false,
                type: 'CLUSTER',
            },
        });
    });
});
