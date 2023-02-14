import { useState, useEffect } from 'react';

import { ClusterForPermissions, getClustersForPermissions } from 'services/RolesService';
import { Cluster } from 'types/cluster.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type Result = {
    isLoading: boolean;
    clusters: Cluster[];
    error: string;
};

function useFetchClustersForPermissions(permissions: string[]): Result {
    const defaultResultState = {
        clusters: [],
        error: '',
        isLoading: true,
    };

    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        getClustersForPermissions(permissions)
            .then((data) => {
                const responseClusters = data.clusters;
                const clusters: Cluster[] = [];
                responseClusters.forEach((responseCluster: ClusterForPermissions) => {
                    const cluster: Cluster = {} as Cluster;
                    cluster.id = responseCluster.id;
                    cluster.name = responseCluster.name;
                    clusters.push(cluster);
                });
                setResult({
                    clusters: clusters || [],
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

export default useFetchClustersForPermissions;
