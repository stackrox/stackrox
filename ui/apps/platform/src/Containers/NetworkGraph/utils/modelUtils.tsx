import { EdgeModel } from '@patternfly/react-topology';

import { NetworkEntityInfo, Node } from 'types/networkFlow.proto';
import { ensureExhaustive } from 'utils/type.utils';
import {
    CustomModel,
    CustomNodeData,
    CustomNodeModel,
    ExternalNodeModel,
    NamespaceData,
    NamespaceNodeModel,
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

function getDataByEntityType(entity: NetworkEntityInfo, policyIds: string[]): CustomNodeData {
    switch (entity.type) {
        case 'DEPLOYMENT':
            return { ...entity, policyIds };
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

export function transformData(nodes: Node[]): CustomModel {
    const dataModel = {
        graph: graphModel,
        nodes: [] as CustomNodeModel[],
        edges: [] as EdgeModel[],
    };

    const namespaceNodes: Record<string, NamespaceNodeModel> = {};
    let externalNode: ExternalNodeModel | null = null;

    nodes.forEach(({ entity, policyIds, outEdges }) => {
        // creating each node and adding to data model
        const node = {
            id: entity.id,
            type: 'node',
            width: 75,
            height: 75,
            label: getNameByEntity(entity),
            data: getDataByEntityType(entity, policyIds),
        } as CustomNodeModel;

        dataModel.nodes.push(node);

        // to group deployments into namespaces
        if (entity.type === 'DEPLOYMENT') {
            const { namespace } = entity.deployment;
            const namespaceNode = namespaceNodes[namespace];
            if (namespaceNode && namespaceNode?.children) {
                namespaceNode?.children.push(entity.id);
            } else {
                const namespaceData: NamespaceData = {
                    collapsible: true,
                    showContextMenu: false,
                    type: 'NAMESPACE',
                };
                namespaceNodes[namespace] = {
                    id: namespace,
                    type: 'group',
                    children: [entity.id],
                    group: true,
                    label: namespace,
                    style: { padding: 15 },
                    data: namespaceData,
                };
            }
        }

        // to group external entities and cidr blocks to external grouping
        if (entity.type === 'EXTERNAL_SOURCE' || entity.type === 'INTERNET') {
            if (!externalNode) {
                externalNode = {
                    id: 'External to cluster',
                    type: 'group',
                    children: [],
                    group: true,
                    label: 'External to cluster',
                    style: { padding: 15 },
                    data: {
                        type: 'EXTERNAL',
                        collapsible: true,
                        showContextMenu: false,
                    },
                };
            }
            if (externalNode && externalNode?.children) {
                externalNode.children.push(entity.id);
            }
        }

        // creating edges based off of outEdges per node and adding to data model
        Object.keys(outEdges).forEach((nodeIdx) => {
            const edge = {
                id: `${entity.id} ${nodes[nodeIdx].entity.id as string}`,
                type: 'edge',
                source: entity.id,
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
