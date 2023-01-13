import { useEffect, useState } from 'react';

import { fetchNetworkBaselines } from 'services/NetworkService';
import { NetworkBaseline } from 'types/networkBaseline.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { Flow } from '../types/flow.type';
import { protocolLabel } from '../utils/flowUtils';

type FetchNetworkBaselinesResult = {
    isLoading: boolean;
    data: { networkBaselines: Flow[]; isAlertingEnabled: boolean };
    error: string | null;
};

const defaultResultState = {
    data: { networkBaselines: [], isAlertingEnabled: false },
    error: null,
    isLoading: true,
};

/*
 * This hook does an API call to the baseline status API to get the baseline status
 * of the supplied peers
 */
function useFetchNetworkBaselines(deploymentId): FetchNetworkBaselinesResult {
    const [result, setResult] = useState<FetchNetworkBaselinesResult>(defaultResultState);

    useEffect(() => {
        const networkBaselinesPromise = fetchNetworkBaselines({ deploymentId });

        networkBaselinesPromise
            .then((response: NetworkBaseline) => {
                const { peers, locked, namespace } = response;
                const isAlertingEnabled = locked;
                const networkBaselines = peers.reduce((acc, currPeer) => {
                    const currPeerType = currPeer.entity.info.type;
                    const type = currPeerType === 'DEPLOYMENT' ? 'Deployment' : 'External';
                    let entity = '';
                    if (currPeerType === 'DEPLOYMENT') {
                        entity = currPeer.entity.info.deployment.name;
                    } else if (currPeerType === 'EXTERNAL_SOURCE') {
                        entity = currPeer.entity.info.externalSource.name;
                    } else if (currPeerType === 'INTERNET') {
                        entity = 'External entities';
                    }
                    // we need a unique id for each network flow
                    const newNetworkBaselines = currPeer.properties.map(
                        ({ ingress, port, protocol }) => {
                            const direction = ingress ? 'Ingress' : 'Egress';
                            const flowId = `${entity}-${namespace}-${direction}-${port}-${protocol}`;
                            const networkBaseline = {
                                id: flowId,
                                type,
                                entity,
                                namespace,
                                direction: ingress ? 'Ingress' : 'Egress',
                                port: String(port),
                                protocol: protocolLabel[protocol],
                                isAnomalous: false,
                                children: [],
                            };
                            return networkBaseline;
                        }
                    );
                    return [...acc, ...newNetworkBaselines] as Flow[];
                }, [] as Flow[]);
                setResult({
                    isLoading: false,
                    data: { networkBaselines, isAlertingEnabled },
                    error: null,
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({
                    isLoading: false,
                    data: { networkBaselines: [], isAlertingEnabled: false },
                    error: errorMessage,
                });
            });
    }, [deploymentId]);

    return result;
}

export default useFetchNetworkBaselines;
