import { useState, useEffect } from 'react';

import { fetchClustersAsArray } from 'services/ClustersService';
import { Cluster } from 'types/cluster.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type Result = {
    isLoading: boolean;
    clusters: Cluster[];
    error: string;
};

const defaultResultState = {
    clusters: [],
    error: '',
    isLoading: true,
};

function useFetchClusters(): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        fetchClustersAsArray()
            .then((data) => {
                setResult({
                    clusters: (data as Cluster[]) || [],
                    error: '',
                    isLoading: false,
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({
                    clusters: [],
                    error: errorMessage,
                    isLoading: false,
                });
            });
    }, []);

    return result;
}

export default useFetchClusters;
