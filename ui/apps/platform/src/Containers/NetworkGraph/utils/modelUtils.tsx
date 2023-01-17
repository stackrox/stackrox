import { EdgeStyle } from '@patternfly/react-topology';

import {
    DeploymentNetworkEntityInfo,
    ExternalSourceNetworkEntityInfo,
    InternetNetworkEntityInfo,
    NetworkEntityInfo,
    Node,
    OutEdges,
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

export function transformActiveData(
    nodes: Node[],
    policyNodeMap: Record<string, DeploymentNodeModel>
): CustomModel {
    const dataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as CustomEdgeModel[],
    };

    const namespaceNodes: Record<string, NamespaceNodeModel> = {};
    const externalNodes: Record<string, ExternalEntitiesNodeModel | CIDRBlockNodeModel> = {};
    const deploymentNodes: Record<string, DeploymentNodeModel> = {};

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
            const edge = {
                id: `${id} ${nodes[nodeIdx].entity.id as string}`,
                type: 'edge',
                source: id,
                target: nodes[nodeIdx].entity.id,
                visible: false,
                data: {
                    properties,
                },
            };
            dataModel.edges.push(edge);
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
        dataModel.nodes.push(externalGroupNode);
    }

    // add deployment nodes to data model
    dataModel.nodes.push(...Object.values(deploymentNodes));

    // add namespace nodes to data model
    dataModel.nodes.push(...Object.values(namespaceNodes));

    // add external nodes to data model
    dataModel.nodes.push(...Object.values(externalNodes));

    return dataModel;
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

export function transformPolicyData(nodes: Node[], flows: number): CustomModel {
    const dataModel: CustomModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as CustomEdgeModel[],
    };
    nodes.forEach(({ entity, policyIds, outEdges, nonIsolatedEgress, nonIsolatedIngress }) => {
        const networkPolicyState = getNetworkPolicyState(nonIsolatedEgress, nonIsolatedIngress);
        const node = getNodeModel(
            entity,
            policyIds,
            networkPolicyState,
            POLICY_NODE_EXTERNALLY_CONNECTED_VALUE,
            outEdges
        );
        dataModel.nodes.push(node);

        // creating edges based off of outEdges per node and adding to data model
        Object.keys(outEdges).forEach((nodeIdx) => {
            const { properties } = outEdges[nodeIdx];
            const edge = {
                id: `${entity.id} ${nodes[nodeIdx].entity.id as string}`,
                type: 'edge',
                source: entity.id,
                target: nodes[nodeIdx].entity.id,
                visible: false,
                edgeStyle: EdgeStyle.dashed,
                data: {
                    properties,
                },
            };
            dataModel.edges.push(edge);
        });
    });
    const { extraneousEgressNode, extraneousIngressNode } = createExtraneousNodes(flows);
    dataModel.nodes.push(extraneousEgressNode);
    dataModel.nodes.push(extraneousIngressNode);
    return dataModel;
}

export function createExtraneousFlowsModel(
    policyDataModel: CustomModel,
    activeNodeMap: Record<string, CustomNodeModel>,
    activeEdgeMap: Record<string, CustomEdgeModel>
): CustomModel {
    const dataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as CustomEdgeModel[],
    };
    const namespaceNodes: Record<string, NamespaceNodeModel> = {};
    let externalNode: ExternalGroupNodeModel | null = null;
    // add all non-group nodes from the active graph
    Object.values(activeNodeMap).forEach((node) => {
        if (!node.group) {
            dataModel.nodes.push(node);
        }
    });

    // loop through each node in policy graph to see if it exists in the active graph
    policyDataModel.nodes?.forEach((node) => {
        // add to extraneous flows model when the node is not in the active graph
        // and is not a group node in the policy graph
        if (!activeNodeMap[node.id] && !node.group) {
            dataModel.nodes.push(node);
        }
    });

    // loop through each edge in policy graph to see if it exists in the active graph
    policyDataModel.edges?.forEach((edge) => {
        // only add to extraneous flows model when edge is not in the active graph
        if (!activeEdgeMap[edge.id]) {
            dataModel.edges.push(edge);
        }
    });

    // TODO: need somewhere to check the currently selected node to see if it is nonIsolated ingress/egress
    // and add the appropriate Egress flows or Ingress flows grouped node accordingly

    // find namespace and external nodes
    dataModel.nodes.forEach(({ data }) => {
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
    dataModel.nodes.push(...Object.values(namespaceNodes));

    // add external group node to data model
    if (externalNode) {
        dataModel.nodes.push(externalNode);
    }

    return dataModel;
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
        },
    };
    return { extraneousEgressEdge, extraneousIngressEdge };
}
