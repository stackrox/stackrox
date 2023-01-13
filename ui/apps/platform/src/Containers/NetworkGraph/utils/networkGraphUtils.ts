import { ListenPort } from 'types/networkFlow.proto';
import { CustomEdgeModel, CustomNodeModel, DeploymentNodeModel } from '../types/topology.type';

/* node helper functions */

export function getDeploymentNodesInNamespace(
    nodes: CustomNodeModel[],
    namespaceId: string
): DeploymentNodeModel[] {
    const namespaceNode = nodes.find((node) => node.id === namespaceId);
    if (!namespaceNode) {
        return [];
    }
    const deploymentNodes = nodes.filter((node) => {
        return namespaceNode.children?.includes(node.id);
    }) as DeploymentNodeModel[];
    return deploymentNodes;
}

function getExternalNodeIds(nodes: CustomNodeModel[]): string[] {
    const externalNodeIds =
        nodes?.reduce((acc, curr) => {
            if (curr.data.type === 'EXTERNAL_ENTITIES' || curr.data.type === 'CIDR_BLOCK') {
                return [...acc, curr.id];
            }
            return acc;
        }, [] as string[]) || [];
    return externalNodeIds;
}

export function getNodeById(
    nodes: CustomNodeModel[] | undefined,
    nodeId: string | undefined
): CustomNodeModel | undefined {
    return nodes?.find((node) => node.id === nodeId);
}

/* edge helper functions */

export function getNumInternalFlows(
    nodes: CustomNodeModel[],
    edges: CustomEdgeModel[],
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
    nodes: CustomNodeModel[],
    edges: CustomEdgeModel[],
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

export function getEdgesByNodeId(edges: CustomEdgeModel[], id: string): CustomEdgeModel[] {
    return edges.filter((edge) => {
        return edge.source === id || edge.target === id;
    });
}

export function getNumDeploymentFlows(edges: CustomEdgeModel[], deploymentId: string): number {
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

export function getListenPorts(nodes: CustomNodeModel[], deploymentId: string): ListenPort[] {
    const deployment = nodes?.find((node) => {
        return node.id === deploymentId;
    });
    if (!deployment || deployment.data.type !== 'DEPLOYMENT') {
        return [];
    }
    return deployment.data.deployment.listenPorts;
}
