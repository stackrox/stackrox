import { useEffect, useState } from 'react';

import { fetchNetworkBaselines } from 'services/NetworkService';
import { filterLabels } from 'constants/networkFilterModes';
import { FlattenedNetworkBaseline } from 'Containers/Network/networkTypes';
import { networkFlowStatus } from 'constants/networkGraph';

type Result = { isLoading: boolean; data: FlattenedNetworkBaseline[]; error: string | null };

/*
 * This hook does an API call to the baseline status API to get the baseline status
 * of the supplied peers
 */
function useFetchNetworkBaselines({ deploymentId, filterState }): Result {
    const [result, setResult] = useState<Result>({ data: [], error: null, isLoading: true });

    useEffect(() => {
        const networkBaselinesPromise = fetchNetworkBaselines({ deploymentId });

        networkBaselinesPromise
            .then((response) => {
                const { deploymentName, namespace, peers } = response;
                const data = peers.reduce((acc, curr) => {
                    curr.properties.forEach((property) => {
                        const peer = {
                            entity: {
                                id: curr.entity.info.id,
                                type: curr.entity.info.type,
                                name: deploymentName,
                                namespace,
                            },
                            ingress: property.ingress,
                            port: property.port,
                            protocol: property.protocol,
                            state: filterLabels[filterState],
                        };
                        acc.push({
                            peer,
                            status: networkFlowStatus.BASELINE,
                        });
                    });
                    return acc;
                }, []);
                setResult({ data, error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ data: [], error, isLoading: false });
            });
    }, [deploymentId, filterState]);

    return result;
}

export default useFetchNetworkBaselines;
