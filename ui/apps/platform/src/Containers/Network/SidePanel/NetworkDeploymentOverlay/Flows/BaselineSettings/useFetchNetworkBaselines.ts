import { useEffect, useState } from 'react';

import { fetchNetworkBaselines } from 'services/NetworkService';
import { BaselineStatus, FlattenedNetworkBaseline } from 'Containers/Network/networkTypes';
import { networkFlowStatus, nodeTypes } from 'constants/networkGraph';

type FetchNetworkBaselinesResult = {
    isLoading: boolean;
    data: { networkBaselines: FlattenedNetworkBaseline[]; isAlertingEnabled: boolean };
    error: string | null;
};

const defaultResultState = {
    data: { networkBaselines: [], isAlertingEnabled: false },
    error: null,
    isLoading: true,
};

export function getPeerEntityName(peer): string {
    switch (peer.entity.info.type) {
        case nodeTypes.EXTERNAL_ENTITIES:
            return 'External Entities';
        case nodeTypes.CIDR_BLOCK:
            return peer.entity.info.externalSource.name as string;
        default:
            return peer.entity.info.deployment.name as string;
    }
}

/*
 * This hook does an API call to the baseline status API to get the baseline status
 * of the supplied peers
 */
function useFetchNetworkBaselines({
    selectedDeployment,
    deploymentId,
    filterState,
    entityIdToNamespaceMap,
}): FetchNetworkBaselinesResult {
    const [result, setResult] = useState<FetchNetworkBaselinesResult>(defaultResultState);

    useEffect(() => {
        const networkBaselinesPromise = fetchNetworkBaselines({ deploymentId });

        networkBaselinesPromise
            .then((response) => {
                const { peers, locked } = response;
                const networkBaselines = peers.reduce(
                    (acc: FlattenedNetworkBaseline[], currPeer) => {
                        currPeer.properties.forEach((property) => {
                            const name = getPeerEntityName(currPeer);
                            const entityId = currPeer.entity.info.id;
                            const namespace = entityIdToNamespaceMap[entityId] || '';
                            const peer = {
                                entity: {
                                    id: entityId,
                                    type: currPeer.entity.info.type,
                                    name,
                                    namespace,
                                },
                                ingress: property.ingress,
                                port: property.port,
                                protocol: property.protocol,
                            };
                            acc.push({
                                peer,
                                status: networkFlowStatus.BASELINE as BaselineStatus,
                            });
                        });
                        return acc;
                    },
                    []
                );
                setResult({
                    data: {
                        networkBaselines,
                        isAlertingEnabled: locked,
                    },
                    error: null,
                    isLoading: false,
                });
            })
            .catch((error) => {
                setResult({
                    data: { networkBaselines: [], isAlertingEnabled: false },
                    error,
                    isLoading: false,
                });
            });
        // TODO: Possibly use another value other than selectedDeployment to ensure this logic
        // is executed again. See following comment: https://github.com/stackrox/stackrox/pull/7254#discussion_r555252326
    }, [selectedDeployment, deploymentId, filterState]);

    return result;
}

export default useFetchNetworkBaselines;
