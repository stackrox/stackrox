import { ListenPort } from 'types/networkFlow.proto';
import { CustomEdgeModel, CustomNodeModel, DeploymentNodeModel } from '../types/topology.type';
import { Flow } from '../types/flow.type';

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

export function getNodeById(
    nodes: CustomNodeModel[] | undefined,
    nodeId: string | undefined
): CustomNodeModel | undefined {
    return nodes?.find((node) => node.id === nodeId);
}

/* edge helper functions */

export function getNumAnomalousInternalFlows(networkFlows: Flow[]) {
    const numAnomalousInternalFlows =
        networkFlows.reduce((acc, flow) => {
            if (
                flow.isAnomalous &&
                flow.type !== 'CIDR_BLOCK' &&
                flow.type !== 'EXTERNAL_ENTITIES'
            ) {
                return acc + 1;
            }
            return acc;
        }, 0) || 0;
    return numAnomalousInternalFlows;
}

export function getNumAnomalousExternalFlows(networkFlows: Flow[]) {
    const numAnomalousExternalFlows =
        networkFlows.reduce((acc, flow) => {
            if (
                flow.isAnomalous &&
                (flow.type === 'CIDR_BLOCK' || flow.type === 'EXTERNAL_ENTITIES')
            ) {
                return acc + 1;
            }
            return acc;
        }, 0) || 0;
    return numAnomalousExternalFlows;
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
