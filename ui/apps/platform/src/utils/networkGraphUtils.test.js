import {
    getSideMap,
    getNamespaceEdges,
    getEdgesFromNode,
    getClasses,
    getNodeData,
    getNamespaceEdgeNodes,
    getActiveNamespaceList,
    getIngressPortsAndProtocols,
    getEgressPortsAndProtocols,
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
            const bundledEdges = getNamespaceEdges(configObj);
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

    describe('getIngressPortsAndProtocols', () => {
        it('should return the edges going to a target node', () => {
            const node = {
                id: '5',
                ingress: ['3', '4'],
                egress: ['1'],
                outEdges: { 0: { properties: [{ port: '8443', protocol: 'L4_PROTOCOL_TCP' }] } },
            };
            const nodes = [
                {
                    id: '1',
                    outEdges: {
                        1: { properties: [{ port: '4000', protocol: 'L4_PROTOCOL_TCP' }] },
                    },
                },
                {
                    id: '2',
                    outEdges: { 3: { properties: [{ port: '9', protocol: 'L4_PROTOCOL_UDP' }] } },
                },
                {
                    id: '3',
                    outEdges: { 4: { properties: [{ port: '443', protocol: 'L4_PROTOCOL_TCP' }] } },
                },
                {
                    id: '4',
                    outEdges: { 4: { properties: [{ port: '53', protocol: 'L4_PROTOCOL_UDP' }] } },
                },
                {
                    id: '5',
                    outEdges: {
                        0: { properties: [{ port: '8443', protocol: 'L4_PROTOCOL_TCP' }] },
                    },
                },
            ];
            const ingressPortsAndProtocols = getIngressPortsAndProtocols(nodes, node);
            expect(ingressPortsAndProtocols).toEqual([
                { port: '443', protocol: 'L4_PROTOCOL_TCP' },
                { port: '53', protocol: 'L4_PROTOCOL_UDP' },
            ]);
        });
    });

    describe('getEgressPortsAndProtocols', () => {
        it('should return the edges going out of a source node', () => {
            const node = {
                id: '5',
                ingress: ['3', '4'],
                egress: ['1'],
                outEdges: { 0: { properties: [{ port: '8443', protocol: 'L4_PROTOCOL_TCP' }] } },
            };
            const egressPortsAndProtocols = getEgressPortsAndProtocols(node.outEdges);
            expect(egressPortsAndProtocols).toEqual([
                { port: '8443', protocol: 'L4_PROTOCOL_TCP' },
            ]);
        });
    });
});
