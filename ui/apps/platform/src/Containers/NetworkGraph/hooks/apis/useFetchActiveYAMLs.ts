import { useEffect, useState } from 'react';
import { fetchNetworkPoliciesByClusterId } from 'services/NetworkService';
import { NetworkPolicy } from 'types/networkPolicy.proto';

type Result = { isLoading: boolean; networkPolicies: NetworkPolicy[]; error: string | null };

const defaultResultState = { networkPolicies: [], error: null, isLoading: true };

type UseFetchActiveYAMLsParams = {
    clusterId: string;
};

function useFetchActiveYAMLs({ clusterId }: UseFetchActiveYAMLsParams): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        fetchNetworkPoliciesByClusterId(clusterId)
            .then((data: NetworkPolicy[]) => {
                setResult({ networkPolicies: data, error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ networkPolicies: [], error, isLoading: false });
            });
    }, [clusterId]);

    return result;
}

export default useFetchActiveYAMLs;
