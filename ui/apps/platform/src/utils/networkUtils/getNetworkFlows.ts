import { filterModes } from 'constants/networkFilterModes';
import { networkTraffic, networkConnections } from 'constants/networkGraph';
import { getIsExternalEntitiesNode, getIsCIDRBlockNode } from 'utils/networkGraphUtils';
import { PortsAndProtocols, Edge } from 'Containers/Network/networkTypes';

// TODO: reconcile this NetworkFlow type with the one in
//       ui/apps/platform/src/Containers/Network/SidePanel/NetworkDeploymentOverlay/NetworkFlows/NetworkFlowsTable/NetworkFlowsTable.tsx
type NetworkFlow = {
    connection: 'active' | 'allowed' | 'active/allowed' | '-';
    deploymentId: string;
    entityName: string;
    namespace: string;
    portsAndProtocols: PortsAndProtocols[];
    traffic: 'bidirectional' | 'ingress' | 'egress';
    type: 'deployment' | 'external';
};

export type OmnibusNetworkFlows = {
    networkFlows: NetworkFlow[];
    numEgressFlows: number;
    numIngressFlows: number;
};

type DirectionalFlowMethods = {
    incrementFlows: (string) => void;
    getNumIngressFlows: () => number;
    getNumEgressFlows: () => number;
};

type NodeMapping = {
    connection: 'active' | 'allowed' | 'active/allowed' | '-';
    deploymentId: string;
    entityName: string;
    namespace: string;
    portsAndProtocols: PortsAndProtocols[];
    traffic: 'bidirectional' | 'ingress' | 'egress';
    type: string;
};

function GetDirectionalFlows(): DirectionalFlowMethods {
    let numIngressFlows = 0;
    let numEgressFlows = 0;
    return {
        incrementFlows: (traffic: string): void => {
            if (traffic === networkTraffic.INGRESS || traffic === networkTraffic.BIDIRECTIONAL) {
                numIngressFlows += 1;
            }
            if (traffic === networkTraffic.EGRESS || traffic === networkTraffic.BIDIRECTIONAL) {
                numEgressFlows += 1;
            }
        },
        getNumIngressFlows: (): number => numIngressFlows,
        getNumEgressFlows: (): number => numEgressFlows,
    };
}

function getConnectionText(filterState, isActive, isAllowed): string {
    let connection = '-';
    const isActiveOrAll = filterState === filterModes.active || filterState === filterModes.all;
    const isAllowedOrAll = filterState === filterModes.allowed || filterState === filterModes.all;
    if (isActiveOrAll && isActive) {
        connection = networkConnections.ACTIVE;
    } else if (isAllowedOrAll && isAllowed) {
        connection = networkConnections.ALLOWED;
    }
    return connection;
}

function getTraffic(portsAndProtocols: PortsAndProtocols[]) {
    const { isIngress, isEgress } = portsAndProtocols.reduce(
        (acc, curr) => {
            if (curr.traffic === networkTraffic.INGRESS) {
                acc.isIngress = true;
            } else if (curr.traffic === networkTraffic.EGRESS) {
                acc.isEgress = true;
            } else if (curr.traffic === networkTraffic.BIDIRECTIONAL) {
                acc.isIngress = true;
                acc.isEgress = true;
            }
            return acc;
        },
        { isIngress: false, isEgress: false }
    );
    if (isIngress && isEgress) {
        return networkTraffic.BIDIRECTIONAL;
    }
    if (isIngress) {
        return networkTraffic.INGRESS;
    }
    if (isEgress) {
        return networkTraffic.EGRESS;
    }
    throw new Error('Network flow should have ports and protocols');
}

/**
 * Grabs the deployment-to-deployment edges and filters based on the filter state
 *
 * @param {!Edges[]} edges
 * @param {!Number} filterState
 * @returns {!OmnibusNetworkFlows}
 */
export function getNetworkFlows(edges: Edge[], filterState): OmnibusNetworkFlows {
    if (!edges) {
        return { networkFlows: [], numIngressFlows: 0, numEgressFlows: 0 };
    }

    let networkFlows;
    const directionalFlows: DirectionalFlowMethods = GetDirectionalFlows();
    const nodeMapping = edges.reduce(
        (
            acc,
            {
                data: {
                    destNodeId,
                    destNodeName,
                    destNodeNamespace,
                    destNodeType,
                    isActive,
                    isAllowed,
                    portsAndProtocols,
                },
            }
        ) => {
            // don't double count edges that are divided because they're within different namespaces
            if (acc[destNodeId]) {
                return acc;
            }
            const isExternal =
                getIsExternalEntitiesNode(destNodeType) || getIsCIDRBlockNode(destNodeType);
            const connection = getConnectionText(filterState, isActive, isAllowed);
            // See https://github.com/stackrox/stackrox/pull/7800/files#r592623997 for explanation of why we are
            // constructing traffic like this instead of from the data object
            const traffic = getTraffic(portsAndProtocols);
            directionalFlows.incrementFlows(traffic);
            return {
                ...acc,
                [destNodeId]: {
                    traffic,
                    deploymentId: destNodeId,
                    entityName: destNodeName,
                    namespace: isExternal ? '-' : destNodeNamespace,
                    type: isExternal ? 'external' : 'deployment',
                    connection,
                    portsAndProtocols,
                    entityType: destNodeType,
                },
            };
        },
        {}
    );
    switch (filterState) {
        case filterModes.active:
            networkFlows = Object.values<NodeMapping>(nodeMapping).filter(
                (value) => value.connection === networkConnections.ACTIVE
            );
            break;
        case filterModes.allowed:
            networkFlows = Object.values<NodeMapping>(nodeMapping).filter(
                (value) => value.connection === networkConnections.ALLOWED
            );
            break;
        default:
            networkFlows = Object.values(nodeMapping);
    }
    const numIngressFlows = directionalFlows.getNumIngressFlows();
    const numEgressFlows = directionalFlows.getNumEgressFlows();
    return { networkFlows, numIngressFlows, numEgressFlows };
}
