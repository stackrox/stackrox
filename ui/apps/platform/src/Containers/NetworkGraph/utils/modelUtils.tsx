import { EdgeModel } from '@patternfly/react-topology';

import { NetworkEntityInfo, Node } from 'types/networkFlow.proto';
import {
    CIDRBlockData,
    CustomModel,
    CustomNodeData,
    CustomNodeModel,
    DeploymentData,
    ExternalEntitiesData,
    ExternalNodeModel,
    NamespaceData,
    NamespaceNodeModel,
} from '../types/topology.type';

function getNameByEntity(entity: NetworkEntityInfo): string {
    const { type } = entity;
    switch (type) {
        case 'DEPLOYMENT':
            return entity.deployment.name;
        case 'INTERNET':
            return 'Internet';
        case 'EXTERNAL_SOURCE':
            return entity.externalSource.name;
        default:
            return '';
    }
}

function getDataByEntityType(entity: NetworkEntityInfo, policyIds: string[]): CustomNodeData {
    if (entity.type === 'DEPLOYMENT') {
        const data: DeploymentData = {
            ...entity,
            policyIds,
        };
        return data;
    }
    if (entity.type === 'INTERNET') {
        const data: ExternalEntitiesData = {
            ...entity,
            type: 'EXTERNAL_ENTITIES',
        };
        return data;
    }
    const data: CIDRBlockData = {
        ...entity,
        type: 'CIDR_BLOCK',
    };
    return data;
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
    const externalNode: ExternalNodeModel = {
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

    // add group nodes to data model
    dataModel.nodes.push(...Object.values(namespaceNodes), externalNode);

    return dataModel;
}
