import { useEffect, useMemo, useState } from 'react';

import { getClustersForPermissions, ScopeObject } from 'services/RolesService';
import { Cluster } from 'types/cluster.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type Result = {
    isLoading: boolean;
    clusters: Cluster[];
    error: string;
};

function useFetchClustersForPermissions(permissions: string[]): Result {
    const defaultResultState = useMemo(() => {
        return {
            clusters: [],
            error: '',
            isLoading: true,
        };
    }, []);

    const [result, setResult] = useState<Result>({
        clusters: [],
        error: '',
        isLoading: false,
    });

    const [requestedPermissions] = useState<string[]>(permissions);

    useEffect(() => {
        setResult(defaultResultState);

        getClustersForPermissions(requestedPermissions)
            .then((data) => {
                const clusters: Cluster[] = data.clusters.map((responseCluster: ScopeObject) => {
                    return {
                        id: responseCluster.id,
                        name: responseCluster.name,
                    } as Cluster;
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
    }, [defaultResultState, requestedPermissions]);

    return result;
}

export default useFetchClustersForPermissions;
