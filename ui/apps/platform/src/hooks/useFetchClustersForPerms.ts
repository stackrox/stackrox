import { useState, useEffect } from 'react';

import { Cluster } from 'types/cluster.proto';
import {
    AccessLevel, ClusterForPermission,
    getClustersForPermission
} from 'services/RolesService';
import { getAxiosErrorMessage } from "../utils/responseErrorUtils";

type Result = {
    isLoading: boolean;
    clusters: Cluster[];
    error: string;
};

function useFetchClustersForPermission(resource: string, access: AccessLevel): Result {
    const defaultResultState = {
        clusters: [],
        error: '',
        isLoading: true,
    };

    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState)

        getClustersForPermission({resource: resource, access: access})
            .then((data) => {
                const responseClusters = data.clusters;
                const clusters: Cluster[] = [];
                responseClusters.forEach((rspCl: ClusterForPermission) => {
                    let cluster : Cluster = {} as Cluster;
                    cluster.id = rspCl.id
                    cluster.name = rspCl.name
                    clusters.push(cluster);
                });
                setResult({
                    clusters: (clusters as Cluster[]) || [],
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

    return result
}

export default useFetchClustersForPermission;