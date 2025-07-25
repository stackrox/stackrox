import { useEffect, useState } from 'react';

import { getClustersForPermissions } from 'services/RolesService';
import type { ClusterScopeObject } from 'services/RolesService';
import type { ResourceName } from 'types/roleResources';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type Result = {
    isLoading: boolean;
    clusters: ClusterScopeObject[];
    error: string;
};

const defaultResultState = {
    clusters: [],
    error: '',
    isLoading: true,
};

function useFetchClustersForPermissions(permissions: ResourceName[]): Result {
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
                    clusters: data?.clusters || [],
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
    }, [requestedPermissions]);

    return result;
}

export default useFetchClustersForPermissions;
