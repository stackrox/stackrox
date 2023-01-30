import { useEffect, useState } from 'react';

import { fetchNetworkPolicies } from 'services/NetworkService';
import { NetworkPolicy } from 'types/networkPolicy.proto';

type Result = { isLoading: boolean; networkPolicies: NetworkPolicy[]; error: string | null };

const defaultResultState = { networkPolicies: [], error: null, isLoading: true };

/*
 * This hook does an API call to the network policies API to get a list of network policies
 */
function useFetchNetworkPolicies(policyIds: string[]): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        if (policyIds) {
            fetchNetworkPolicies(policyIds)
                .then((data) => {
                    setResult({
                        networkPolicies: data?.response as NetworkPolicy[],
                        error: null,
                        isLoading: false,
                    });
                })
                .catch((error) => {
                    setResult({ networkPolicies: [], error, isLoading: false });
                });
        }

        return () => setResult(defaultResultState);
    }, [policyIds]);

    return result;
}

export default useFetchNetworkPolicies;
