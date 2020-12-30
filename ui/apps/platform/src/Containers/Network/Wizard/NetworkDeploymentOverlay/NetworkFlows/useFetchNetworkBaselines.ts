import { useEffect, useState } from 'react';

import { getNetworkFlows } from 'utils/networkUtils/getNetworkFlows';
import { fetchNetworkBaselineStatus } from 'services/NetworkService';
import {
    BaselineStatus,
    FlattenedPeer,
    NetworkFlow,
    NetworkBaseline,
    FlattenedNetworkBaseline,
} from 'Containers/Network/Wizard/NetworkDeploymentOverlay/NetworkFlows/networkTypes';

/*
 * This function will unflatten the flattened network baselines by grouping
 * the network baselines by ports and protocols
 *
 */
function unflattenNetworkBaseline(
    flattenedNetworkBaseline: FlattenedNetworkBaseline
): NetworkBaseline {
    const { peer, status } = flattenedNetworkBaseline;
    const networkBaseline = {
        peer: {
            entity: {
                id: peer.entity.id,
                type: peer.entity.type,
                name: peer.entity.name,
                namespace: peer.entity.namespace,
            },
            portsAndProtocols: [
                { port: peer.port, protocol: peer.protocol, ingress: peer.ingress },
            ],
            ingress: peer.ingress,
            egress: !peer.ingress,
            state: peer.state,
        },
        status,
    };
    return networkBaseline;
}

/*
 * This function will unflatten the flattened network baselines by grouping
 * the network baselines by ports and protocols
 *
 */
function unflattenNetworkBaselines(
    networkBaselines: FlattenedNetworkBaseline[]
): NetworkBaseline[] {
    const groupMap = networkBaselines.reduce(
        (acc: { [key: string]: NetworkBaseline }, curr: FlattenedNetworkBaseline) => {
            if (!acc[curr.peer.entity.id]) {
                const datum = unflattenNetworkBaseline(curr);
                acc[curr.peer.entity.id] = datum;
            } else {
                const datum = { ...acc[curr.peer.entity.id] };
                datum.peer.portsAndProtocols.push({
                    port: curr.peer.port,
                    protocol: curr.peer.protocol,
                    ingress: curr.peer.ingress,
                });
                // when both ingress and egress are true, it is bidirectional
                if (datum.peer.ingress !== curr.peer.ingress) {
                    datum.peer.ingress = true;
                    datum.peer.egress = true;
                }
                acc[curr.peer.entity.id] = datum;
            }
            return acc;
        },
        {}
    );
    const unflattenedNetworkBaselines = Object.values(groupMap);
    return unflattenedNetworkBaselines;
}

/*
 * This function takes the network flows and separates them based on their ports
 * and protocols
 */
function flattenNetworkFlows(networkFlows): NetworkFlow[] {
    return networkFlows.reduce((acc, curr) => {
        curr.portsAndProtocols.forEach(({ port, protocol, traffic }) => {
            const datum = { ...curr, port, protocol, traffic };
            delete datum.portsAndProtocols;
            acc.push(datum);
        });
        return acc;
    }, []);
}

/*
 * This function creates the peer object used for the baseline status API
 */
function createPeerFromNetworkFlow(networkFlow): FlattenedPeer {
    const peer = {
        entity: {
            id: networkFlow.deploymentId,
            type: networkFlow.entityType,
            name: networkFlow.entityName,
            namespace: networkFlow.namespace,
        },
        ingress: networkFlow.traffic === 'ingress',
        port: networkFlow.port,
        protocol: networkFlow.protocol,
        state: networkFlow.connection,
    };
    return peer;
}

/*
 * This function creates the peers based on flattening out the network flows
 * to be used for the baseline status API call
 */
function getPeersFromNetworkFlows(networkFlows): FlattenedPeer[] {
    const flattenedNetworkFlows = flattenNetworkFlows(networkFlows);
    return flattenedNetworkFlows.map((networkFlow) => {
        const peer = createPeerFromNetworkFlow(networkFlow);
        return peer;
    });
}

/*
 * This function creates a unique key based on the fields of a peer
 */
function getBaselineStatusKey({ id, ingress, port, protocol }): string {
    return `${id}-${ingress}-${port}-${protocol}`;
}

/*
 * This hook does an API call to the baseline status API to get the baseline status
 * of the supplied peers
 */
function useFetchNetworkBaselines({
    deploymentId,
    edges,
    filterState,
}): { isLoading: boolean; networkBaselines: NetworkBaseline[] } {
    const [networkBaselines, setNetworkBaselines] = useState<NetworkBaseline[]>([]);
    const [isLoading, setLoading] = useState(true);

    useEffect(() => {
        const { networkFlows } = getNetworkFlows(edges, filterState);
        const peers = getPeersFromNetworkFlows(networkFlows);
        const baselineStatusPromise = fetchNetworkBaselineStatus({ deploymentId, peers });

        baselineStatusPromise.then((response) => {
            const baselineStatusMap: { [key: string]: BaselineStatus } = response.statuses.reduce(
                (acc, networkBaseline: FlattenedNetworkBaseline) => {
                    const key = getBaselineStatusKey({
                        id: networkBaseline.peer.entity.id,
                        ingress: networkBaseline.peer.ingress,
                        port: networkBaseline.peer.port,
                        protocol: networkBaseline.peer.protocol,
                    });
                    acc[key] = networkBaseline.status;
                    return acc;
                },
                {}
            );
            const flattenedNetworkBaselines = peers.reduce(
                (acc: FlattenedNetworkBaseline[], peer: FlattenedPeer) => {
                    const key = getBaselineStatusKey({
                        id: peer.entity.id,
                        ingress: peer.ingress,
                        port: peer.port,
                        protocol: peer.protocol,
                    });
                    const status = baselineStatusMap[key];
                    acc.push({
                        peer,
                        status,
                    });
                    return acc;
                },
                []
            );
            const unflattenedNetworkBaselines = unflattenNetworkBaselines(
                flattenedNetworkBaselines
            );
            setNetworkBaselines(unflattenedNetworkBaselines);
            setLoading(false);
        });
    }, [deploymentId, edges, filterState]);

    return { isLoading, networkBaselines };
}

export default useFetchNetworkBaselines;
