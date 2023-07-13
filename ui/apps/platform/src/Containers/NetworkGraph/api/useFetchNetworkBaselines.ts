import { useEffect, useState } from 'react';
import uniqBy from 'lodash/uniqBy';

import { fetchNetworkBaselines } from 'services/NetworkService';
import { NetworkBaseline } from 'types/networkBaseline.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { Flow, FlowEntityType } from '../types/flow.type';

type Result = {
    isLoading: boolean;
    data: { networkBaselines: Flow[]; isAlertingOnBaselineViolation: boolean };
    error: string | null;
};

type FetchNetworkBaselinesResult = {
    refetchBaselines: () => void;
} & Result;

const defaultResultState = {
    data: { networkBaselines: [], isAlertingOnBaselineViolation: false },
    error: null,
    isLoading: true,
};

/*
 * This hook does an API call to the baseline status API to get the baseline status
 * of the supplied peers
 */
function useFetchNetworkBaselines(deploymentId): FetchNetworkBaselinesResult {
    const [result, setResult] = useState<Result>(defaultResultState);

    function fetchBaselines() {
        fetchNetworkBaselines({ deploymentId })
            .then((response: NetworkBaseline) => {
                const { peers, locked, namespace } = response;
                const isAlertingOnBaselineViolation = locked;
                const networkBaselines = peers.reduce((acc, currPeer) => {
                    const currPeerType = currPeer.entity.info.type;
                    const entityId = currPeer.entity.info.id;
                    let entity = '';
                    let type: FlowEntityType = 'DEPLOYMENT';
                    if (currPeerType === 'DEPLOYMENT') {
                        entity = currPeer.entity.info.deployment.name;
                        type = 'DEPLOYMENT';
                    } else if (currPeerType === 'EXTERNAL_SOURCE') {
                        entity = currPeer.entity.info.externalSource.name;
                        type = 'CIDR_BLOCK';
                    } else if (currPeerType === 'INTERNET') {
                        entity = 'External entities';
                        type = 'EXTERNAL_ENTITIES';
                    }
                    // we need a unique id for each network flow
                    const newNetworkBaselines = currPeer.properties.map(
                        ({ ingress, port, protocol }) => {
                            const direction = ingress ? 'Ingress' : 'Egress';
                            const flowId = `${entity}-${namespace}-${direction}-${port}-${protocol}`;
                            const networkBaseline: Flow = {
                                id: flowId,
                                type,
                                entity,
                                entityId,
                                namespace,
                                direction: ingress ? 'Ingress' : 'Egress',
                                port: String(port),
                                protocol,
                                isAnomalous: false,
                                children: [],
                            };
                            return networkBaseline;
                        }
                    );
                    return [...acc, ...newNetworkBaselines];
                }, [] as Flow[]);
                const uniqNetworkBaselines = uniqBy(networkBaselines, 'id');
                setResult({
                    isLoading: false,
                    data: { networkBaselines: uniqNetworkBaselines, isAlertingOnBaselineViolation },
                    error: null,
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({
                    isLoading: false,
                    data: { networkBaselines: [], isAlertingOnBaselineViolation: false },
                    error: errorMessage,
                });
            });
    }

    useEffect(() => {
        fetchBaselines();

        return () => setResult(defaultResultState);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [deploymentId]);

    return { ...result, refetchBaselines: fetchBaselines };
}

export default useFetchNetworkBaselines;
