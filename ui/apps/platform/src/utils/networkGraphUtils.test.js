import { filterModes } from 'constants/networkFilterModes';
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
    getNetworkFlows,
} from './networkGraphUtils';
import {
    filteredData,
    nodeSideMap,
    configObj,
    namespaceEdges,
    namespaceList,
    deploymentList,
    namespaceEdgeNodes,
    deploymentEdges,
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
            const { networkFlows } = getNetworkFlows(deploymentEdges, filterModes.all);
            const ingressPortsAndProtocols = getIngressPortsAndProtocols(networkFlows);
            expect(ingressPortsAndProtocols).toEqual([
                {
                    port: 123,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'ingress',
                },
                {
                    port: 678,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'ingress',
                },
            ]);
        });
    });

    describe('getEgressPortsAndProtocols', () => {
        it('should return the edges going out of a source node', () => {
            const { networkFlows } = getNetworkFlows(deploymentEdges, filterModes.all);
            const egressPortsAndProtocols = getEgressPortsAndProtocols(networkFlows);
            expect(egressPortsAndProtocols).toEqual([
                {
                    port: 456,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'egress',
                },
                {
                    port: 911,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'egress',
                },
            ]);
        });
    });

    describe('getNetworkFlows', () => {
        it('should return all network flows', () => {
            const { networkFlows } = getNetworkFlows(deploymentEdges, filterModes.all);

            expect(networkFlows).toEqual([
                {
                    connection: 'active',
                    deploymentId: '1',
                    deploymentName: 'node-1',
                    namespace: 'namespace-a',
                    portsAndProtocols: [
                        {
                            port: 123,
                            protocol: 'L4_PROTOCOL_TCP',
                            traffic: 'ingress',
                        },
                    ],
                    traffic: 'ingress',
                },
                {
                    connection: 'allowed',
                    deploymentId: '2',
                    deploymentName: 'node-2',
                    namespace: 'namespace-a',
                    portsAndProtocols: [
                        {
                            port: 456,
                            protocol: 'L4_PROTOCOL_TCP',
                            traffic: 'egress',
                        },
                    ],
                    traffic: 'egress',
                },
                {
                    connection: 'active',
                    deploymentId: '3',
                    deploymentName: 'node-3',
                    namespace: 'namespace-a',
                    portsAndProtocols: [
                        {
                            port: 678,
                            protocol: 'L4_PROTOCOL_TCP',
                            traffic: 'ingress',
                        },
                        {
                            port: 911,
                            protocol: 'L4_PROTOCOL_TCP',
                            traffic: 'egress',
                        },
                    ],
                    traffic: 'bidirectional',
                },
            ]);
        });

        it('should return active network flows', () => {
            const { networkFlows } = getNetworkFlows(deploymentEdges, filterModes.active);

            expect(networkFlows).toEqual([
                {
                    connection: 'active',
                    deploymentId: '1',
                    deploymentName: 'node-1',
                    namespace: 'namespace-a',
                    portsAndProtocols: [
                        {
                            port: 123,
                            protocol: 'L4_PROTOCOL_TCP',
                            traffic: 'ingress',
                        },
                    ],
                    traffic: 'ingress',
                },
                {
                    connection: 'active',
                    deploymentId: '3',
                    deploymentName: 'node-3',
                    namespace: 'namespace-a',
                    portsAndProtocols: [
                        {
                            port: 678,
                            protocol: 'L4_PROTOCOL_TCP',
                            traffic: 'ingress',
                        },
                        {
                            port: 911,
                            protocol: 'L4_PROTOCOL_TCP',
                            traffic: 'egress',
                        },
                    ],
                    traffic: 'bidirectional',
                },
            ]);
        });

        it('should return allowed network flows', () => {
            const { networkFlows } = getNetworkFlows(deploymentEdges, filterModes.allowed);

            expect(networkFlows).toEqual([
                {
                    connection: 'allowed',
                    deploymentId: '2',
                    deploymentName: 'node-2',
                    namespace: 'namespace-a',
                    portsAndProtocols: [
                        {
                            port: 456,
                            protocol: 'L4_PROTOCOL_TCP',
                            traffic: 'egress',
                        },
                    ],
                    traffic: 'egress',
                },
            ]);
        });

        it('should return the correct number of directional flows', () => {
            const { numIngressFlows, numEgressFlows } = getNetworkFlows(
                deploymentEdges,
                filterModes.all
            );

            expect(numIngressFlows).toEqual(2);
            expect(numEgressFlows).toEqual(2);
        });
    });
});
