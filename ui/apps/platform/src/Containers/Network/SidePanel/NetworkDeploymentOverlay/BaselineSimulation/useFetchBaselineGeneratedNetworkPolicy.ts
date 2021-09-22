import { useEffect, useState } from 'react';

import { fetchBaselineGeneratedNetworkPolicy } from 'services/NetworkService';
import { NetworkPolicyModification } from 'Containers/Network/networkTypes';

export type UseFetchBaselineGeneratedNetworkPolicy = {
    deploymentId: string;
    includePorts: boolean;
};

export type FetchBaselineGeneratedNetworkPolicyResult = {
    isGeneratingNetworkPolicy: boolean;
    data: {
        modification: NetworkPolicyModification;
    } | null;
    error: string | null;
};

const defaultResultState = {
    data: null,
    error: null,
    isGeneratingNetworkPolicy: false,
};

function useFetchBaselineGeneratedNetworkPolicy({
    deploymentId,
    includePorts,
}: UseFetchBaselineGeneratedNetworkPolicy): FetchBaselineGeneratedNetworkPolicyResult {
    const [result, setResult] =
        useState<FetchBaselineGeneratedNetworkPolicyResult>(defaultResultState);

    useEffect(() => {
        setResult((prevResult) => ({ ...prevResult, isGeneratingNetworkPolicy: true }));
        fetchBaselineGeneratedNetworkPolicy({ deploymentId, includePorts })
            .then((response) => {
                setResult({
                    data: response,
                    error: null,
                    isGeneratingNetworkPolicy: false,
                });
            })
            .catch((error) => {
                setResult({
                    data: null,
                    error,
                    isGeneratingNetworkPolicy: false,
                });
            });
    }, [deploymentId, includePorts]);

    return result;
}

export default useFetchBaselineGeneratedNetworkPolicy;
