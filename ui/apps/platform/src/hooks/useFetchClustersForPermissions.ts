import { useEffect, useMemo, useState } from 'react';

import { getClustersForPermissions, ClusterScopeObject } from 'services/RolesService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type Result = {
    isLoading: boolean;
    clusters: ClusterScopeObject[];
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
                setResult({
                    clusters: data.clusters,
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
