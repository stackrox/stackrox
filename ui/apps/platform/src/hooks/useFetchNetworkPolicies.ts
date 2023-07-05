import { useState } from 'react';
import useDeepCompareEffect from 'use-deep-compare-effect';

import { fetchNetworkPolicies } from 'services/NetworkService';
import { NetworkPolicy } from 'types/networkPolicy.proto';

type Result = {
    isLoading: boolean;
    networkPolicies: NetworkPolicy[];
    // This error array represents errors that occurred while fetching the individual network policies when
    // the overall request chain succeeded
    networkPolicyErrors: Error[];
    // This single error represents a unrecoverable error that occurred after the overall request chain failed
    error: Error | null;
};

const defaultResultState = {
    networkPolicies: [],
    networkPolicyErrors: [],
    error: null,
    isLoading: true,
};

/*
 * This hook does an API call to the network policies API to get a list of network policies
 */
function useFetchNetworkPolicies(policyIds: string[]): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useDeepCompareEffect(() => {
        setResult(defaultResultState);

        if (policyIds) {
            fetchNetworkPolicies(policyIds)
                .then(({ policies, errors }) => {
                    setResult({
                        networkPolicies: policies,
                        networkPolicyErrors: errors,
                        error: null,
                        isLoading: false,
                    });
                })
                .catch((error) => {
                    setResult({
                        networkPolicies: [],
                        networkPolicyErrors: [],
                        error,
                        isLoading: false,
                    });
                });
        }

        return () => setResult(defaultResultState);
    }, [policyIds]);

    return result;
}

export default useFetchNetworkPolicies;
