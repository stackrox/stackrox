import { useState } from 'react';

import { fetchBaselineGeneratedNetworkPolicy } from 'services/NetworkService';
import { NetworkPolicyModification } from 'types/networkPolicy.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type FetchBaselineNetworkPolicyParams = {
    deploymentId: string;
    includePorts: boolean;
};

export type Result = {
    isLoading: boolean;
    data: {
        modification: NetworkPolicyModification | null;
    };
    error: string;
};

type FetchBaselineNetworkPolicyResult = {
    fetchBaselineNetworkPolicy: (
        onSuccessCallback: (modification: NetworkPolicyModification) => void
    ) => void;
} & Result;

const defaultResultState = {
    isLoading: false,
    data: { modification: null },
    error: '',
};

function useFetchBaselineNetworkPolicy({
    deploymentId,
    includePorts,
}: FetchBaselineNetworkPolicyParams): FetchBaselineNetworkPolicyResult {
    const [result, setResult] = useState<Result>(defaultResultState);

    function fetchBaselineNetworkPolicy(onSuccessCallback) {
        setResult({ data: { modification: null }, isLoading: true, error: '' });
        fetchBaselineGeneratedNetworkPolicy({ deploymentId, includePorts })
            .then((response) => {
                setResult({
                    isLoading: false,
                    data: response,
                    error: '',
                });
                onSuccessCallback(response?.modification);
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({
                    isLoading: false,
                    data: { modification: null },
                    error: errorMessage,
                });
            });
    }

    return { ...result, fetchBaselineNetworkPolicy };
}

export default useFetchBaselineNetworkPolicy;
