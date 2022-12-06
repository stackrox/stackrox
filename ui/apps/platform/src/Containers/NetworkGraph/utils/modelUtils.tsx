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
    PolicyNodeModel,
    ExtraneousNodeModel,
    NetworkPolicyState,
} from '../types/topology.type';

function getNameByEntity(entity: NetworkEntityInfo): string {
    switch (entity.type) {
        case 'DEPLOYMENT':
            return entity.deployment.name;
        case 'INTERNET':
            return 'Internet';
        case 'EXTERNAL_SOURCE':
            return entity.externalSource.name;
        default:
            return ensureExhaustive(entity);
    }
}

function getNetworkPolicyState(policyNode?: PolicyNodeModel): NetworkPolicyState {
    let networkPolicyState: NetworkPolicyState = 'none';
    if (policyNode) {
        const { nonIsolatedEgress, nonIsolatedIngress } = policyNode.data;
        if (!nonIsolatedIngress && !nonIsolatedEgress) {
            networkPolicyState = 'both';
        } else if (nonIsolatedEgress) {
            networkPolicyState = 'ingress';
        } else if (nonIsolatedIngress) {
            networkPolicyState = 'egress';
        }
    }
    return networkPolicyState;
}

function getDataByEntityType(
    entity: NetworkEntityInfo,
    policyIds: string[],
    policyNode?: PolicyNodeModel
): CustomNodeData {
    switch (entity.type) {
        case 'DEPLOYMENT':
            return { ...entity, policyIds, networkPolicyState: getNetworkPolicyState(policyNode) };
        case 'INTERNET':
            return { ...entity, type: 'EXTERNAL_ENTITIES' };
        case 'EXTERNAL_SOURCE':
            return { ...entity, type: 'CIDR_BLOCK' };
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

function getNodeModel(
    entity: NetworkEntityInfo,
    policyIds: string[],
    policyNode?: PolicyNodeModel
): CustomNodeModel {
    const node = {
        id: entity.id,
        type: 'node',
        width: 75,
        height: 75,
        label: getNameByEntity(entity),
    } as CustomNodeModel;
    node.data = getDataByEntityType(entity, policyIds, policyNode);
    return node;
}

export function transformActiveData(
    nodes: Node[],
    policyNodeMap: Record<string, PolicyNodeModel>
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
        // creating each node and adding to data model
        const node = getNodeModel(entity, policyIds, policyNodeMap[id]);

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

export function transformPolicyData(nodes: Node[]): CustomModel {
    const dataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as EdgeModel[],
    };
    nodes.forEach(({ entity, policyIds, outEdges }) => {
        const node = getNodeModel(entity, policyIds);
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

export function getExtraneousNodes(): {
    extraneousEgressNode: ExtraneousNodeModel;
    extraneousIngressNode: ExtraneousNodeModel;
} {
    const extraneousEgressNode: ExtraneousNodeModel = {
        id: 'extraneous-egress',
        type: 'fakeGroup',
        width: 75,
        height: 75,
        label: 'Egress flows',
        // TODO: figure out how to fake group node
        // group: true,
        // children: [],
        data: {
            collapsible: false,
            showContextMenu: false,
            type: 'EXTRANEOUS',
        },
    };
    const extraneousIngressNode: ExtraneousNodeModel = {
        id: 'extraneous-ingress',
        type: 'fakeGroup',
        width: 75,
        height: 75,
        label: 'Ingress flows',
        // TODO: figure out how to fake group node
        // group: true,
        // children: [],
        data: {
            collapsible: false,
            showContextMenu: false,
            type: 'EXTRANEOUS',
        },
    };
    return { extraneousEgressNode, extraneousIngressNode };
}
