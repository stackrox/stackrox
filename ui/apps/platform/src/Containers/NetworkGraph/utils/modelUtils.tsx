import { EdgeModel, EdgeStyle } from '@patternfly/react-topology';

import { NetworkEntityInfo, Node } from 'types/networkFlow.proto';
import { ensureExhaustive } from 'utils/type.utils';
import {
    CustomModel,
    CustomNodeData,
    CustomNodeModel,
    ExternalNodeModel,
    ExternalData,
    NamespaceData,
    NamespaceNodeModel,
    DeploymentNodeModel,
    ExtraneousNodeModel,
    NetworkPolicyState,
} from '../types/topology.type';

function getNameByEntity(entity: NetworkEntityInfo): string {
    switch (entity.type) {
        case 'DEPLOYMENT':
            return entity.deployment.name;
        case 'INTERNET':
            return 'External Entities';
        case 'EXTERNAL_SOURCE':
            return entity.externalSource.name;
        default:
            return ensureExhaustive(entity);
    }
}

export const graphModel = {
    id: 'stackrox-active-graph',
    type: 'graph',
    layout: 'ColaGroups',
};

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

function getExternalGroupNode(): ExternalNodeModel {
    const externalGroupData: ExternalData = {
        collapsible: true,
        showContextMenu: false,
        type: 'EXTERNAL',
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

function getDataByEntityType(
    entity: NetworkEntityInfo,
    policyIds: string[],
    networkPolicyState: NetworkPolicyState
): CustomNodeData {
    switch (entity.type) {
        case 'DEPLOYMENT':
            return {
                ...entity,
                policyIds,
                networkPolicyState,
                showPolicyState: true,
            };
        case 'INTERNET':
            return { ...entity, type: 'EXTERNAL_ENTITIES' };
        case 'EXTERNAL_SOURCE':
            return { ...entity, type: 'CIDR_BLOCK' };
        default:
            return ensureExhaustive(entity);
    }
}

function getNodeModel(
    entity: NetworkEntityInfo,
    policyIds: string[],
    networkPolicyState: NetworkPolicyState
): CustomNodeModel {
    const node = {
        id: entity.id,
        type: 'node',
        width: 75,
        height: 75,
        label: getNameByEntity(entity),
    } as CustomNodeModel;
    node.data = getDataByEntityType(entity, policyIds, networkPolicyState);
    return node;
}

export function transformActiveData(
    nodes: Node[],
    policyNodeMap: Record<string, DeploymentNodeModel>
): CustomModel {
    const dataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as EdgeModel[],
    };

    const namespaceNodes: Record<string, NamespaceNodeModel> = {};
    let externalNode: ExternalNodeModel | null = null;

    nodes.forEach(({ entity, policyIds, outEdges }) => {
        const { type, id } = entity;
        const { networkPolicyState } = policyNodeMap[id]?.data || {};
        // creating each node and adding to data model
        const node = getNodeModel(entity, policyIds, networkPolicyState);

        dataModel.nodes.push(node);

        // to group deployments into namespaces
        if (type === 'DEPLOYMENT') {
            const { namespace } = entity.deployment;
            const namespaceNode = namespaceNodes[namespace];
            if (namespaceNode && namespaceNode?.children) {
                namespaceNode?.children.push(id);
            } else {
                namespaceNodes[namespace] = getNamespaceNode(namespace, id);
            }
        }

        // to group external entities and cidr blocks to external grouping
        if (type === 'EXTERNAL_SOURCE' || type === 'INTERNET') {
            if (!externalNode) {
                externalNode = getExternalGroupNode();
            }
            if (externalNode && externalNode?.children) {
                externalNode.children.push(id);
            }
        }

        // creating edges based off of outEdges per node and adding to data model
        Object.keys(outEdges).forEach((nodeIdx) => {
            const edge = {
                id: `${id} ${nodes[nodeIdx].entity.id as string}`,
                type: 'edge',
                source: id,
                target: nodes[nodeIdx].entity.id,
                visible: false,
            };
            dataModel.edges.push(edge);
        });
    });

    // add namespace nodes to data model
    dataModel.nodes.push(...Object.values(namespaceNodes));

    // add external group node to data model
    if (externalNode) {
        dataModel.nodes.push(externalNode);
    }

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

export function transformPolicyData(nodes: Node[], flows: number): CustomModel {
    const dataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as EdgeModel[],
    };
    nodes.forEach(({ entity, policyIds, outEdges, nonIsolatedEgress, nonIsolatedIngress }) => {
        const networkPolicyState = getNetworkPolicyState(nonIsolatedEgress, nonIsolatedIngress);
        const node = getNodeModel(entity, policyIds, networkPolicyState);
        dataModel.nodes.push(node);

        // creating edges based off of outEdges per node and adding to data model
        Object.keys(outEdges).forEach((nodeIdx) => {
            const edge = {
                id: `${entity.id} ${nodes[nodeIdx].entity.id as string}`,
                type: 'edge',
                source: entity.id,
                target: nodes[nodeIdx].entity.id,
                visible: false,
                edgeStyle: EdgeStyle.dashed,
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
    activeEdgeMap: Record<string, EdgeModel>
): CustomModel {
    const dataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as EdgeModel[],
    };
    const namespaceNodes: Record<string, NamespaceNodeModel> = {};
    let externalNode: ExternalNodeModel | null = null;
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
    extraneousEgressEdge: EdgeModel;
    extraneousIngressEdge: EdgeModel;
} {
    const extraneousEgressEdge = {
        id: 'extraneous-egress-edge',
        type: 'edge',
        source: selectedNodeId,
        target: 'extraneous-egress',
        visible: true,
        edgeStyle: EdgeStyle.dashed,
    };
    const extraneousIngressEdge = {
        id: 'extraneous-ingress-edge',
        type: 'edge',
        source: 'extraneous-ingress',
        target: selectedNodeId,
        visible: true,
        edgeStyle: EdgeStyle.dashed,
    };
    return { extraneousEgressEdge, extraneousIngressEdge };
}
