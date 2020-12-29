import { filterModes } from 'constants/networkFilterModes';
import { getNetworkFlows } from 'utils/networkUtils/getNetworkFlows';
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
    createPortsAndProtocolsSelector,
    getNodeName,
    getNodeNamespace,
    getIsNodeHoverable,
    getExternalEntitiesEdgeNodes,
    getCIDRBlockEdgeNodes,
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
} from './networkGraphUtils.test.constants.ts';

const nodeTypes = {
    DEPLOYMENT: 'DEPLOYMENT',
    EXTERNAL_ENTITIES: 'INTERNET',
    CIDR_BLOCK: 'EXTERNAL_SOURCE',
};

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
            expect(bundledEdges.length).toEqual(namespaceEdges.length);
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
                    port: 111,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'ingress',
                },
                {
                    port: 444,
                    protocol: 'L4_PROTOCOL_UDP',
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
                    port: 222,
                    protocol: 'L4_PROTOCOL_UDP',
                    traffic: 'egress',
                },
                {
                    port: 333,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'egress',
                },
                {
                    port: 555,
                    protocol: 'L4_PROTOCOL_TCP',
                    traffic: 'egress',
                },
            ]);
        });
    });

    describe('createPortsAndProtocolsSelector', () => {
        it('should get ports/protocols when it exists in the mapping', () => {
            const nodes = [
                {
                    entity: { type: 'DEPLOYMENT', id: '0' },
                    outEdges: {
                        1: { properties: [{ port: 8443, protocol: 'L4_PROTOCOL_TCP' }] },
                        2: { properties: [{ port: 3000, protocol: 'L4_PROTOCOL_TCP' }] },
                    },
                },
                {
                    entity: { type: 'DEPLOYMENT', id: '1' },
                    outEdges: {},
                },
                {
                    entity: { type: 'DEPLOYMENT', id: '2' },
                    outEdges: {},
                },
            ];
            const highlightedNodeId = nodes[0].entity.type.id;
            const networkNodeMap = {
                0: {
                    active: {
                        entity: { type: 'DEPLOYMENT', id: '0' },
                        outEdges: {
                            1: { properties: [{ port: 8443, protocol: 'L4_PROTOCOL_TCP' }] },
                            2: { properties: [{ port: 3000, protocol: 'L4_PROTOCOL_TCP' }] },
                        },
                    },
                    egressAllowed: [1, 2],
                    egressActive: [],
                },
                1: {
                    active: {
                        entity: { type: 'DEPLOYMENT', id: '1' },
                        outEdges: {},
                    },
                    egressAllowed: [],
                    egressActive: [],
                },
                2: {
                    active: {
                        entity: { type: 'DEPLOYMENT', id: '2' },
                        outEdges: {},
                    },
                    egressAllowed: [],
                    egressActive: [],
                },
            };
            const filterState = filterModes.active;

            const getPortsAndProtocolsByLink = createPortsAndProtocolsSelector(
                nodes,
                highlightedNodeId,
                networkNodeMap,
                filterState
            );

            expect(getPortsAndProtocolsByLink('0**__**1')).toEqual([
                { port: 8443, protocol: 'L4_PROTOCOL_TCP', traffic: 'egress' },
            ]);
        });

        it('should get a single element that represents any protocol/any ports when it does not exist in the mapping', () => {
            const nodes = [
                {
                    entity: { type: 'DEPLOYMENT', id: '0' },
                    outEdges: {
                        1: { properties: [{ port: 8443, protocol: 'L4_PROTOCOL_TCP' }] },
                        2: { properties: [{ port: 3000, protocol: 'L4_PROTOCOL_TCP' }] },
                    },
                },
                {
                    entity: { type: 'DEPLOYMENT', id: '1' },
                    outEdges: {},
                },
                {
                    entity: { type: 'DEPLOYMENT', id: '2' },
                    outEdges: {},
                },
            ];
            const highlightedNodeId = nodes[0].entity.type.id;
            const isEgress = false;

            const getPortsAndProtocolsByLink = createPortsAndProtocolsSelector(
                nodes,
                highlightedNodeId
            );

            expect(getPortsAndProtocolsByLink('1**__**0', isEgress)).toEqual([
                { port: '*', protocol: 'L4_PROTOCOL_ANY', traffic: 'ingress' },
            ]);
        });
    });

    describe('getNodeNamespace', () => {
        it('should get the namespace value for a deployment node', () => {
            const node = {
                entity: {
                    id: '1234',
                    type: 'DEPLOYMENT',
                    deployment: {
                        namespace: 'N1',
                    },
                },
            };
            expect(getNodeNamespace(node)).toEqual('N1');
        });

        it('should get the namespace value for an external entities node', () => {
            const node = {
                entity: {
                    id: '1234',
                    type: nodeTypes.EXTERNAL_ENTITIES,
                },
            };
            expect(getNodeNamespace(node)).toEqual('1234');
        });

        it('should get the namespace value for a CIDR block node', () => {
            const node = {
                entity: {
                    id: '1234',
                    type: nodeTypes.CIDR_BLOCK,
                },
            };
            expect(getNodeNamespace(node)).toEqual('1234');
        });

        it('should throw an error when an unexpected type is supplied', () => {
            const node = {
                entity: {
                    id: '1234',
                    type: 'UNKNOWN',
                },
            };
            function getUnknownNodeNamespace() {
                return getNodeNamespace(node);
            }
            expect(getUnknownNodeNamespace).toThrowError(
                'Node with unexpected type (UNKNOWN) was supplied to function'
            );
        });
    });

    describe('getNodeName', () => {
        it('should get the name value for a deployment node', () => {
            const node = {
                entity: {
                    id: '1234',
                    type: 'DEPLOYMENT',
                    deployment: {
                        name: 'D1',
                    },
                },
            };
            expect(getNodeName(node)).toEqual('D1');
        });

        it('should get the name value for an external entities node', () => {
            const node = {
                entity: {
                    id: '1234',
                    type: nodeTypes.EXTERNAL_ENTITIES,
                },
            };
            expect(getNodeName(node)).toEqual('External Entities');
        });

        it('should get the name value for a CIDR block node', () => {
            const node = {
                entity: {
                    id: '1234',
                    type: nodeTypes.CIDR_BLOCK,
                    externalSource: {
                        name: 'Amazon us-east-1',
                        cidr: '10.10.0.1/24',
                    },
                },
            };
            expect(getNodeName(node)).toEqual(
                `${node.entity.externalSource.cidr} / ${node.entity.externalSource.name}`
            );
        });

        it('should throw an error when an unexpected type is supplied', () => {
            const node = {
                entity: {
                    id: '1234',
                    type: 'UNKNOWN',
                },
            };
            function getUnknownNodeName() {
                return getNodeName(node);
            }
            expect(getUnknownNodeName).toThrowError(
                'Node with unexpected type (UNKNOWN) was supplied to function'
            );
        });
    });

    describe('getIsNodeHoverable', () => {
        it('should be a hoverable node', () => {
            expect(getIsNodeHoverable(nodeTypes.DEPLOYMENT)).toEqual(true);
            expect(getIsNodeHoverable('INTERNET')).toEqual(true);
            expect(getIsNodeHoverable(nodeTypes.CIDR_BLOCK)).toEqual(true);
        });

        it('should not be a hoverable node', () => {
            expect(getIsNodeHoverable('NAMESPACE')).toEqual(false);
        });
    });

    describe('getExternalEntitiesEdgeNodes', () => {
        it('should get the edge nodes for the external entities node', () => {
            const externalEntityNode = { data: { id: '1' } };
            const externalEntityEdgeNodes = [
                {
                    data: { id: '1_top', parent: '1', side: 'top' },
                    classes: 'externalEntitiesEdge',
                },
                {
                    data: { id: '1_left', parent: '1', side: 'left' },
                    classes: 'externalEntitiesEdge',
                },
                {
                    data: { id: '1_right', parent: '1', side: 'right' },
                    classes: 'externalEntitiesEdge',
                },
                {
                    data: { id: '1_bottom', parent: '1', side: 'bottom' },
                    classes: 'externalEntitiesEdge',
                },
            ];
            expect(getExternalEntitiesEdgeNodes(externalEntityNode)).toEqual(
                externalEntityEdgeNodes
            );
        });
    });

    describe('getCIDRBlockEdgeNodes', () => {
        it('should get the edge nodes for all CIDR block nodes', () => {
            const cidrBlockNodes = [{ data: { id: '1' } }, { data: { id: '2' } }];
            const cidrBlockEdgeNodes = [
                { data: { id: '1_top', parent: '1', side: 'top' }, classes: 'cidrBlockEdge' },
                {
                    data: { id: '1_left', parent: '1', side: 'left' },
                    classes: 'cidrBlockEdge',
                },
                {
                    data: { id: '1_right', parent: '1', side: 'right' },
                    classes: 'cidrBlockEdge',
                },
                {
                    data: { id: '1_bottom', parent: '1', side: 'bottom' },
                    classes: 'cidrBlockEdge',
                },
                { data: { id: '2_top', parent: '2', side: 'top' }, classes: 'cidrBlockEdge' },
                {
                    data: { id: '2_left', parent: '2', side: 'left' },
                    classes: 'cidrBlockEdge',
                },
                {
                    data: { id: '2_right', parent: '2', side: 'right' },
                    classes: 'cidrBlockEdge',
                },
                {
                    data: { id: '2_bottom', parent: '2', side: 'bottom' },
                    classes: 'cidrBlockEdge',
                },
            ];
            expect(getCIDRBlockEdgeNodes(cidrBlockNodes)).toEqual(cidrBlockEdgeNodes);
        });
    });
});
