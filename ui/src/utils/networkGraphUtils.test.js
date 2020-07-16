import {
    getSideMap,
    getNamespaceEdges,
    getEdgesFromNode,
    getClasses,
    getNodeData,
    getNamespaceEdgeNodes,
    getActiveNamespaceList,
} from './networkGraphUtils';
import {
    filteredData,
    nodeSideMap,
    configObj,
    namespaceEdges,
    namespaceList,
    deploymentList,
    namespaceEdgeNodes,
} from './networkGraphUtils.test.constants';

describe('networkGraphUtils', () => {
    describe('getSideMap', () => {
        it('should return closest side when given a source and target', () => {
            const side = getSideMap('stackrox', 'kube-system', nodeSideMap);

            expect(side).toEqual({
                distance: 294,
                source: 'stackrox_left',
                sourceSide: 'left',
                target: 'kube-system_right',
                targetSide: 'right',
            });
        });
    });

    describe('getNamespaceEdges', () => {
        it('should return bundled edges between namespaces', () => {
            const bundledEdges = getNamespaceEdges(null, configObj);
            expect(bundledEdges).toEqual(namespaceEdges);
        });
    });

    describe('getEdgesFromNode', () => {
        it('should return edges for a specific node', () => {
            const edgesFromNode = getEdgesFromNode(
                '6ff5049d-b70a-11ea-a716-025000000001',
                configObj
            );
            expect(edgesFromNode).toEqual([]);
        });
    });

    describe('getClasses', () => {
        it('should return a string of classes for a given map', () => {
            const classes = getClasses({
                active: true,
                deployment: true,
            });
            expect(classes).toEqual('active deployment');
        });
    });

    describe('getNodeData', () => {
        it('should return specified deployment by id', () => {
            const testDeployment = getNodeData(deploymentList[0].data.deploymentId, deploymentList);
            expect(testDeployment).toEqual([deploymentList[0]]);
        });
    });

    describe('getActiveNamespaceList', () => {
        it('should return namespaces that contain active deployments', () => {
            const testActiveNSList = getActiveNamespaceList(filteredData, deploymentList);
            expect(testActiveNSList).toEqual([]);
        });
    });

    describe('getNamespaceEdgeNodes', () => {
        it('should return edge nodes for namespace boxes', () => {
            const testNSEdgeNodes = getNamespaceEdgeNodes(namespaceList);
            expect(testNSEdgeNodes).toEqual(namespaceEdgeNodes);
        });
    });
});
