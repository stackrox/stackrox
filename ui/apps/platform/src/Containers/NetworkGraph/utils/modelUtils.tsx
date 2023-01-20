import { EdgeStyle, EdgeTerminalType } from '@patternfly/react-topology';

import {
    DeploymentNetworkEntityInfo,
    ExternalSourceNetworkEntityInfo,
    InternetNetworkEntityInfo,
    NetworkEntityInfo,
    Node,
    OutEdges,
    L4Protocol,
    EdgeProperties,
} from 'types/networkFlow.proto';
import { ensureExhaustive } from 'utils/type.utils';
import {
    CustomModel,
    CustomNodeModel,
    ExternalGroupNodeModel,
    ExternalGroupData,
    NamespaceData,
    NamespaceNodeModel,
    DeploymentNodeModel,
    ExtraneousNodeModel,
    NetworkPolicyState,
    ExternalEntitiesNodeModel,
    CIDRBlockNodeModel,
    CustomEdgeModel,
} from '../types/topology.type';
import { protocolLabel } from './flowUtils';

export const graphModel = {
    id: 'stackrox-active-graph',
    type: 'graph',
    layout: 'ColaGroups',
};

function getBaseNode(id: string): CustomNodeModel {
    return {
        id,
        type: 'node',
        width: 75,
        height: 75,
    } as CustomNodeModel;
}

function getNamespaceNode(namespace: string, deploymentId: string): NamespaceNodeModel {
    const namespaceData: NamespaceData = {
        collapsible: true,
        showContextMenu: false,
        type: 'NAMESPACE',
    };
    return {
        id: namespace,
        type: 'group',
        children: [deploymentId],
        group: true,
        label: namespace,
        style: { padding: 15 },
        data: namespaceData,
    };
}

function getExternalGroupNode(): ExternalGroupNodeModel {
    const externalGroupData: ExternalGroupData = {
        collapsible: true,
        showContextMenu: false,
        type: 'EXTERNAL_GROUP',
    };
    return {
        id: 'External to cluster',
        type: 'group',
        children: [],
        group: true,
        label: 'External to cluster',
        style: { padding: 15 },
        data: externalGroupData,
    };
}

function getDeploymentNodeModel(
    entity: DeploymentNetworkEntityInfo,
    policyIds: string[],
    networkPolicyState: NetworkPolicyState,
    isExternallyConnected: boolean
): DeploymentNodeModel {
    const baseNode = getBaseNode(entity.id) as DeploymentNodeModel;
    return {
        ...baseNode,
        label: entity.deployment.name,
        data: {
            ...entity,
            policyIds,
            networkPolicyState,
            showPolicyState: true,
            isExternallyConnected,
            showExternalState: true,
        },
    };
}

function getExternalNodeModel(
    entity: ExternalSourceNetworkEntityInfo | InternetNetworkEntityInfo,
    outEdges: OutEdges
): ExternalEntitiesNodeModel | CIDRBlockNodeModel {
    const baseNode = getBaseNode(entity.id);
    switch (entity.type) {
        case 'INTERNET':
            return {
                ...baseNode,
                label: 'External Entities',
                data: { ...entity, type: 'EXTERNAL_ENTITIES', outEdges },
            };
        case 'EXTERNAL_SOURCE':
            return {
                ...baseNode,
                label: entity.externalSource.name,
                data: { ...entity, type: 'CIDR_BLOCK', outEdges },
            };
        default:
            return ensureExhaustive(entity);
    }
}

function getNodeModel(
    entity: NetworkEntityInfo,
    policyIds: string[],
    networkPolicyState: NetworkPolicyState,
    isExternallyConnected: boolean,
    outEdges: OutEdges
): CustomNodeModel {
    switch (entity.type) {
        case 'DEPLOYMENT':
            return getDeploymentNodeModel(
                entity,
                policyIds,
                networkPolicyState,
                isExternallyConnected
            );
        case 'EXTERNAL_SOURCE':
        case 'INTERNET':
            return getExternalNodeModel(entity, outEdges);
        default:
            return ensureExhaustive(entity);
    }
}

function getPortProtocolLabel(port: number, protocol: L4Protocol) {
    return `${port} ${protocolLabel[protocol]}`;
}

export function getPortProtocolEdgeLabel(properties: EdgeProperties[]): string {
    const { port, protocol } = properties[0];
    const singlePortLabel = getPortProtocolLabel(port, protocol);
    return `${properties.length === 1 ? singlePortLabel : properties.length}`;
}

function mergePortProtocolEdgeLabels(firstLabel: string, secondLabel = ''): string {
    const firstLabelNum = Number(firstLabel);
    const secondLabelNum = Number(secondLabel);
    // if both labels are not numbers and current edge label is the same as the previous label,
    // return same label else return 2
    if (!firstLabelNum && !secondLabelNum) {
        return firstLabel === secondLabel ? firstLabel : '2';
    }

    // if the previous label is singular (not a number) but the current edge label is multiple
    // return the sum
    if (!firstLabelNum && !!secondLabelNum) {
        return `${1 + secondLabelNum}`;
    }

    // if current label is singular (not a number) but the prev label is multiple
    // return the sum
    if (!!firstLabelNum && !secondLabelNum) {
        return `${firstLabelNum + 1}`;
    }

    // else return the sum
    return `${firstLabelNum + secondLabelNum}`;
}

export function transformActiveData(
    nodes: Node[],
    policyNodeMap: Record<string, DeploymentNodeModel>
): {
    activeDataModel: CustomModel;
    activeEdgeMap: Record<string, CustomEdgeModel>;
    activeNodeMap: Record<string, CustomNodeModel>;
} {
    const activeDataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as CustomEdgeModel[],
        updateCount: 0,
    };

    const namespaceNodes: Record<string, NamespaceNodeModel> = {};
    const externalNodes: Record<string, ExternalEntitiesNodeModel | CIDRBlockNodeModel> = {};
    const deploymentNodes: Record<string, DeploymentNodeModel> = {};
    const activeEdgeMap: Record<string, CustomEdgeModel> = {};

    nodes.forEach(({ entity, policyIds, outEdges }) => {
        const { type, id } = entity;
        const { networkPolicyState } = policyNodeMap[id]?.data || {};
        const isExternallyConnected = Object.keys(outEdges).some((nodeIdx) => {
            const { entity: targetEntity } = nodes[nodeIdx];
            return targetEntity.type === 'EXTERNAL_SOURCE' || targetEntity.type === 'INTERNET';
        });

        // to group deployments into namespaces
        if (type === 'DEPLOYMENT') {
            const { namespace } = entity.deployment;
            const namespaceNode = namespaceNodes[namespace];
            if (namespaceNode && namespaceNode?.children) {
                namespaceNode?.children.push(id);
            } else {
                namespaceNodes[namespace] = getNamespaceNode(namespace, id);
            }

            // creating deployment nodes
            const deploymentNode = getDeploymentNodeModel(
                entity,
                policyIds,
                networkPolicyState,
                isExternallyConnected
            );

            deploymentNodes[id] = deploymentNode;
        }

        // to group external entities and cidr blocks to external grouping
        if (type === 'EXTERNAL_SOURCE' || type === 'INTERNET') {
            const externalNode = getExternalNodeModel(entity, outEdges);
            if (!externalNodes[id]) {
                externalNodes[id] = externalNode;
            }
        }

        // creating edges based off of outEdges per node and adding to data model
        Object.keys(outEdges).forEach((nodeIdx) => {
            const { properties } = outEdges[nodeIdx];
            const source = id;
            const target: string = nodes[nodeIdx].entity.id;
            const edgeId = `${source}-${target}`;
            const reverseEdgeId = `${target}-${source}`;
            const portProtocolLabel = getPortProtocolEdgeLabel(properties);
            const edge: CustomEdgeModel = {
                id: edgeId,
                type: 'edge',
                source,
                target,
                visible: false,
                data: {
                    tag: portProtocolLabel,
                    portProtocolLabel,
                    properties,
                    isBidirectional: false,
                },
            };
            // this is to reuse the first edge if the edge is bidirectional;
            if (activeEdgeMap[reverseEdgeId]) {
                edge.id = reverseEdgeId;
                edge.data.startTerminalType = EdgeTerminalType.directional;
                edge.data.endTerminalType = EdgeTerminalType.directional;
                const mergedPortEdgeLabel = mergePortProtocolEdgeLabels(
                    portProtocolLabel,
                    activeEdgeMap[reverseEdgeId].data.tag
                );
                edge.data.tag = mergedPortEdgeLabel;
                edge.data.portProtocolLabel = mergedPortEdgeLabel;
                edge.data.isBidirectional = true;
                activeEdgeMap[reverseEdgeId] = edge;
            } else {
                activeEdgeMap[edgeId] = edge;
            }
        });
    });

    const externalNodeIds = Object.keys(externalNodes);
    if (externalNodeIds.length > 0) {
        // adding external outEdges to nodes to indicate externally connected state
        Object.values(externalNodes).forEach((externalNode) => {
            const { outEdges } = externalNode.data || {};
            Object.keys(outEdges).forEach((nodeIdx) => {
                const { id: targetNodeId } = nodes[nodeIdx];
                if (deploymentNodes[targetNodeId]) {
                    deploymentNodes[targetNodeId].data.isExternallyConnected = true;
                }
            });
        });
        // add external group node to data model
        const externalGroupNode: ExternalGroupNodeModel = getExternalGroupNode();
        externalNodeIds.forEach((externalNodeId) => {
            if (externalGroupNode?.children) {
                externalGroupNode.children.push(externalNodeId);
            }
        });

        // add external group node to data model
        activeDataModel.nodes.push(externalGroupNode);
    }

    // add deployment nodes to data model
    activeDataModel.nodes.push(...Object.values(deploymentNodes));

    // add namespace nodes to data model
    activeDataModel.nodes.push(...Object.values(namespaceNodes));

    // add external nodes to data model
    activeDataModel.nodes.push(...Object.values(externalNodes));

    // add edges to data model
    activeDataModel.edges.push(...Object.values(activeEdgeMap));
    return {
        activeDataModel,
        // set activeEdgeMap to be able to cross reference edges by id for the extraneous graph
        activeEdgeMap,
        // set activeNodeMap to be able to cross reference nodes by id for the extraneous graph
        activeNodeMap: { ...deploymentNodes, ...externalNodes },
    };
}

function getNetworkPolicyState(
    nonIsolatedEgress: boolean,
    nonIsolatedIngress: boolean
): NetworkPolicyState {
    let networkPolicyState: NetworkPolicyState = 'none';

    if (!nonIsolatedIngress && !nonIsolatedEgress) {
        networkPolicyState = 'both';
    } else if (nonIsolatedEgress) {
        networkPolicyState = 'ingress';
    } else if (nonIsolatedIngress) {
        networkPolicyState = 'egress';
    }
    return networkPolicyState;
}

// external connections can only be active, so this is hard coded to false
const POLICY_NODE_EXTERNALLY_CONNECTED_VALUE = false;

export function transformPolicyData(
    nodes: Node[],
    flows: number
): { policyDataModel: CustomModel; policyNodeMap: Record<string, DeploymentNodeModel> } {
    const policyDataModel: CustomModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as CustomEdgeModel[],
        updateCount: 0,
    };
    // set policyNodeMap to be able to cross reference nodes by id to enhance active node data
    const policyNodeMap: Record<string, DeploymentNodeModel> = {};
    // to reference edges so we don't double merge bidirectional edges
    const policyEdgeMap: Record<string, CustomEdgeModel> = {};
    nodes.forEach(({ entity, policyIds, outEdges, nonIsolatedEgress, nonIsolatedIngress }) => {
        const networkPolicyState = getNetworkPolicyState(nonIsolatedEgress, nonIsolatedIngress);
        const node = getNodeModel(
            entity,
            policyIds,
            networkPolicyState,
            POLICY_NODE_EXTERNALLY_CONNECTED_VALUE,
            outEdges
        );
        if (!policyNodeMap[node.id]) {
            policyNodeMap[node.id] = node as DeploymentNodeModel;
        }
        policyDataModel.nodes.push(node);

        // creating edges based off of outEdges per node and adding to data model
        Object.keys(outEdges).forEach((nodeIdx) => {
            const source = entity.id;
            const target: string = nodes[nodeIdx].entity.id;
            const edgeId = `${source}-${target}`;
            const reverseEdgeId = `${target}-${source}`;
            const { properties } = outEdges[nodeIdx];
            const portProtocolLabel = getPortProtocolEdgeLabel(properties);
            const edge: CustomEdgeModel = {
                id: edgeId,
                type: 'edge',
                source,
                target,
                visible: false,
                edgeStyle: EdgeStyle.dashed,
                data: {
                    tag: portProtocolLabel,
                    portProtocolLabel,
                    properties,
                    isBidirectional: false,
                },
            };
            // this is to reuse the first edge if the edge is bidirectional;
            if (policyEdgeMap[reverseEdgeId]) {
                edge.id = reverseEdgeId;
                edge.data.startTerminalType = EdgeTerminalType.directional;
                edge.data.endTerminalType = EdgeTerminalType.directional;
                const mergedPortEdgeLabel = mergePortProtocolEdgeLabels(
                    portProtocolLabel,
                    policyEdgeMap[reverseEdgeId].data.tag
                );
                edge.data.tag = mergedPortEdgeLabel;
                edge.data.portProtocolLabel = mergedPortEdgeLabel;
                edge.data.isBidirectional = true;
                policyEdgeMap[reverseEdgeId] = edge;
            } else {
                policyEdgeMap[edgeId] = edge;
            }
        });
    });
    const { extraneousEgressNode, extraneousIngressNode } = createExtraneousNodes(flows);
    policyDataModel.nodes.push(extraneousEgressNode);
    policyDataModel.nodes.push(extraneousIngressNode);
    policyDataModel.edges.push(...Object.values(policyEdgeMap));
    return { policyDataModel, policyNodeMap };
}

export function createExtraneousFlowsModel(
    policyDataModel: CustomModel,
    activeNodeMap: Record<string, CustomNodeModel>,
    activeEdgeMap: Record<string, CustomEdgeModel>
): CustomModel {
    const extraneousDataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as CustomEdgeModel[],
        updateCount: 0,
    };
    const namespaceNodes: Record<string, NamespaceNodeModel> = {};
    let externalNode: ExternalGroupNodeModel | null = null;
    // add all non-group nodes from the active graph
    Object.values(activeNodeMap).forEach((node) => {
        if (!node.group) {
            extraneousDataModel.nodes.push(node);
        }
    });

    // loop through each node in policy graph to see if it exists in the active graph
    policyDataModel.nodes?.forEach((node) => {
        // add to extraneous flows model when the node is not in the active graph
        // and is not a group node in the policy graph
        if (!activeNodeMap[node.id] && !node.group) {
            extraneousDataModel.nodes.push(node);
        }
    });

    // loop through each edge in policy graph to see if it exists in the active graph
    // only add to extraneous flows model when policy edge is not in the active graph
    policyDataModel.edges?.forEach((policyEdge) => {
        const { id: policyEdgeId, source, target } = policyEdge;
        const reversePolicyEdgeId = `${target as string}-${source as string}`;
        const activeEdge = activeEdgeMap[policyEdgeId];
        const activeReverseEdge = activeEdgeMap[reversePolicyEdgeId];

        if (activeEdge || activeReverseEdge) {
            const existingActiveEdge = activeEdge || activeReverseEdge;
            const { isBidirectional: policyEdgeIsBidirectional } = policyEdge.data;
            const { isBidirectional: activeEdgeIsBidirectional } = existingActiveEdge.data;
            if (policyEdgeIsBidirectional && activeEdgeIsBidirectional) {
                // if both policy and active edges are bidirectional, skip edge
            } else if (policyEdgeIsBidirectional && !activeEdgeIsBidirectional) {
                // if policy edge has both directions and active edge does not, add
                // other direction to extraneous flows model
                let edgeDirection = policyEdgeId;
                if (policyEdgeId === existingActiveEdge.id) {
                    // if the policy edge id matches the existing active edge id,
                    // we need to add the reverse edge id to the extraneous data model
                    edgeDirection = reversePolicyEdgeId;
                }
                const edge = {
                    ...policyEdge,
                    id: edgeDirection,
                    data: {
                        ...policyEdge.data,
                        isBidirectional: false,
                    },
                };
                extraneousDataModel.edges.push(edge);
            } else if (!policyEdgeIsBidirectional && activeEdgeIsBidirectional) {
                // if policy edge has one direction and active edge has both, skip
            }
        } else if (!activeEdge && !activeReverseEdge) {
            // if neither direction exists in the active edge map, add to extraneous edges
            extraneousDataModel.edges.push(policyEdge);
        }
    });

    // find namespace and external nodes
    extraneousDataModel.nodes.forEach(({ data }) => {
        const { type } = data;
        // to group deployments into namespaces
        if (type === 'DEPLOYMENT') {
            const { deployment, id } = data;
            const { namespace } = deployment;
            const namespaceNode = namespaceNodes[namespace];
            if (namespaceNode && namespaceNode?.children) {
                namespaceNode?.children.push(id);
            } else {
                namespaceNodes[namespace] = getNamespaceNode(namespace, id);
            }
        }

        // to group external entities and cidr blocks to external grouping
        if (type === 'EXTERNAL_ENTITIES' || type === 'CIDR_BLOCK') {
            if (!externalNode) {
                externalNode = getExternalGroupNode();
            }
            if (externalNode && externalNode?.children) {
                externalNode.children.push(data.id);
            }
        }
    });

    // add namespace nodes to data model
    extraneousDataModel.nodes.push(...Object.values(namespaceNodes));

    // add external group node to data model
    if (externalNode) {
        extraneousDataModel.nodes.push(externalNode);
    }

    return extraneousDataModel;
}

export function createExtraneousNodes(numFlows: number): {
    extraneousEgressNode: ExtraneousNodeModel;
    extraneousIngressNode: ExtraneousNodeModel;
} {
    const extraneousEgressNode: ExtraneousNodeModel = {
        id: 'extraneous-egress',
        type: 'fakeGroup',
        width: 75,
        height: 75,
        label: 'Egress flows',
        visible: false,
        data: {
            collapsible: false,
            showContextMenu: false,
            type: 'EXTRANEOUS',
            numFlows,
        },
    };
    const extraneousIngressNode: ExtraneousNodeModel = {
        id: 'extraneous-ingress',
        type: 'fakeGroup',
        width: 75,
        height: 75,
        label: 'Ingress flows',
        visible: false,
        data: {
            collapsible: false,
            showContextMenu: false,
            type: 'EXTRANEOUS',
            numFlows,
        },
    };
    return { extraneousEgressNode, extraneousIngressNode };
}

export function createExtraneousEdges(selectedNodeId: string): {
    extraneousEgressEdge: CustomEdgeModel;
    extraneousIngressEdge: CustomEdgeModel;
} {
    const extraneousEgressEdge = {
        id: 'extraneous-egress-edge',
        type: 'edge',
        source: selectedNodeId,
        target: 'extraneous-egress',
        visible: true,
        edgeStyle: EdgeStyle.dashed,
        data: {
            properties: [],
            isBidirectional: false,
            portProtocolLabel: '',
        },
    };
    const extraneousIngressEdge = {
        id: 'extraneous-ingress-edge',
        type: 'edge',
        source: 'extraneous-ingress',
        target: selectedNodeId,
        visible: true,
        edgeStyle: EdgeStyle.dashed,
        data: {
            properties: [],
            isBidirectional: false,
            portProtocolLabel: '',
        },
    };
    return { extraneousEgressEdge, extraneousIngressEdge };
}
