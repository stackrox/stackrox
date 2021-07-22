import { useEffect, useState } from 'react';

import { getNetworkFlows } from 'utils/networkUtils/getNetworkFlows';
import { fetchNetworkBaselineStatuses } from 'services/NetworkService';
import {
    BaselineStatus,
    FlattenedPeer,
    NetworkFlow,
    FlattenedNetworkBaseline,
    Edge,
} from 'Containers/Network/networkTypes';

type Result = { isLoading: boolean; data: FlattenedNetworkBaseline[]; error: string | null };

const defaultResultState = { data: [], error: null, isLoading: true };

/*
 * This function takes the network flows and separates them based on their ports
 * and protocols
 */
function flattenNetworkFlows(networkFlows): NetworkFlow[] {
    return networkFlows.reduce((acc: NetworkFlow[], curr) => {
        curr.portsAndProtocols.forEach(({ port, protocol, traffic }) => {
            const datum = { ...curr, port, protocol, traffic };
            delete datum.portsAndProtocols;
            acc.push(datum);
        });
        return acc;
    }, []) as NetworkFlow[];
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
    // TODO remove type casts when peer argument has a type.
    return `${id as string}-${String(ingress as boolean)}-${port as number}-${protocol as string}`;
}

/*
 * This hook does an API call to the baseline status API to get the baseline status
 * of the supplied peers
 */
function useFetchNetworkBaselines({
    deploymentId,
    edges,
    filterState,
    lastUpdatedTimestamp,
}: {
    deploymentId: string;
    edges: Edge[];
    filterState: number;
    lastUpdatedTimestamp: string;
}): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        const { networkFlows } = getNetworkFlows(edges, filterState);
        const peers = getPeersFromNetworkFlows(networkFlows);
        const peersToSend = peers.filter((peer) => peer.port !== '*');

        fetchNetworkBaselineStatuses({
            deploymentId,
            peers: peersToSend,
        })
            .then((response) => {
                const baselineStatusMap: Record<string, BaselineStatus> = response.statuses.reduce(
                    (
                        acc: Record<string, BaselineStatus>,
                        networkBaseline: FlattenedNetworkBaseline
                    ) => {
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
                const flattenedNetworkBaselines = peers.reduce((acc, peer) => {
                    const key = getBaselineStatusKey({
                        id: peer.entity.id,
                        ingress: peer.ingress,
                        port: peer.port,
                        protocol: peer.protocol,
                    });
                    const status = baselineStatusMap[key] || 'ANOMALOUS';
                    acc.push({
                        peer,
                        status,
                    });
                    return acc;
                }, [] as FlattenedNetworkBaseline[]);
                setResult({ data: flattenedNetworkBaselines || [], error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ data: [], error, isLoading: false });
            });
        // I added this eslint disable because having "edges" causes some extra renders
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [deploymentId, filterState, lastUpdatedTimestamp]);

    return result;
}

export default useFetchNetworkBaselines;
