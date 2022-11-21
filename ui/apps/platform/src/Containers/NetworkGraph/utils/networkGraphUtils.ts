import { EdgeModel, NodeModel } from '@patternfly/react-topology';

import { ListenPort } from 'types/networkFlow.proto';

/* node helper functions */

export function getDeploymentNodesInNamespace(nodes: NodeModel[], namespaceId: string) {
    const namespaceNode = nodes.find((node) => node.id === namespaceId);
    if (!namespaceNode) {
        return [];
    }
    const deploymentNodes = nodes.filter((node) => namespaceNode.children?.includes(node.id));
    return deploymentNodes;
}

function getExternalNodeIds(nodes: NodeModel[]): string[] {
    const externalNodeIds =
        nodes?.reduce((acc, curr) => {
            if (curr.data.type === 'INTERNET' || curr.data.type === 'EXTERNAL_SOURCE') {
                return [...acc, curr.id];
            }
            return acc;
        }, [] as string[]) || [];
    return externalNodeIds;
}

export function getNodeById(
    nodes: NodeModel[] | undefined,
    nodeId: string | undefined
): NodeModel | undefined {
    return nodes?.find((node) => node.id === nodeId);
}

/* edge helper functions */

export function getNumInternalFlows(
    nodes: NodeModel[],
    edges: EdgeModel[],
    deploymentId: string
): number {
    const externalNodeIds = getExternalNodeIds(nodes);
    const numInternalFlows =
        edges?.reduce((acc, edge) => {
            if (
                (edge.source === deploymentId && !externalNodeIds.includes(edge.target || '')) ||
                (edge.target === deploymentId && !externalNodeIds.includes(edge.source || ''))
            ) {
                return acc + 1;
            }
            return acc;
        }, 0) || 0;
    return numInternalFlows;
}

export function getNumExternalFlows(
    nodes: NodeModel[],
    edges: EdgeModel[],
    deploymentId: string
): number {
    const externalNodeIds = getExternalNodeIds(nodes);
    const numExternalFlows =
        edges?.reduce((acc, edge) => {
            if (
                (edge.source === deploymentId && externalNodeIds.includes(edge.target || '')) ||
                (edge.target === deploymentId && externalNodeIds.includes(edge.source || ''))
            ) {
                return acc + 1;
            }
            return acc;
        }, 0) || 0;
    return numExternalFlows;
}

export function getNumDeploymentFlows(edges: EdgeModel[], deploymentId: string): number {
    const numFlows =
        edges?.reduce((acc, edge) => {
            if (edge.source === deploymentId || edge.target === deploymentId) {
                return acc + 1;
            }
            return acc;
        }, 0) || 0;
    return numFlows;
}

/* deployment helper functions */

export function getListenPorts(nodes: NodeModel[], deploymentId: string): ListenPort[] {
    const deployment = nodes?.find((node) => {
        return node.id === deploymentId;
    });
    if (!deployment) {
        return [];
    }
    return deployment.data.deployment.listenPorts as ListenPort[];
}
