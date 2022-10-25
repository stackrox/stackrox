import { Model, NodeModel, EdgeModel } from '@patternfly/react-topology';

import { NetworkEntityInfo, Node } from 'types/networkFlow.proto';

export function getLabel(entity: NetworkEntityInfo): string {
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

export function transformData(nodes: Node[]): Model {
    const dataModel = {
        graph: {
            id: 'stackrox-active-graph',
            type: 'graph',
            layout: 'ColaGroups',
        },
        nodes: [] as NodeModel[],
        edges: [] as EdgeModel[],
    };
    const groupNodes = {};
    nodes.forEach(({ entity, outEdges }) => {
        // creating each node and adding to data model
        const node = {
            id: entity.id,
            type: 'node',
            width: 75,
            height: 75,
            label: getLabel(entity),
        };
        dataModel.nodes.push(node);

        // to group deployments into namespaces
        if (entity.type === 'DEPLOYMENT') {
            const { namespace } = entity.deployment;
            if (groupNodes[namespace]) {
                groupNodes[namespace].children.push(entity.id);
            } else {
                groupNodes[namespace] = {
                    id: namespace,
                    type: 'group',
                    children: [entity.id],
                    group: true,
                    label: namespace,
                    style: { padding: 15 },
                    data: {
                        collapsible: true,
                        showContextMenu: false,
                    },
                };
            }
        }

        // creating edges based off of outEdges per node and adding to data model
        Object.keys(outEdges).forEach((nodeIdx) => {
            const edge = {
                id: `${entity.id} ${nodes[nodeIdx].entity.id as string}`,
                type: 'edge',
                source: entity.id,
                target: nodes[nodeIdx].entity.id,
            };
            dataModel.edges.push(edge);
        });
    });

    // add group nodes to data model
    dataModel.nodes.push(...Object.values(groupNodes));
    return dataModel;
}
