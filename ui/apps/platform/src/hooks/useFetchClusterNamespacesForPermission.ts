import { useState, useEffect } from 'react';

import { fetchDeploymentsCount } from 'services/DeploymentsService';
import {
    AccessLevel,
    NamespaceForPermission,
    getNamespacesForClusterAndPermission,
} from 'services/RolesService';
import { RestSearchOption } from 'services/searchOptionsToQuery';
import { getAxiosErrorMessage } from '../utils/responseErrorUtils';

export type Namespace = {
    metadata: {
        id: string;
        name: string;
    };
    deploymentCount;
};

type Result = {
    isLoading: boolean;
    error: string;
    namespaces: Namespace[];
};

function useFetchNamespacesForClusterAndPermission(
    resource: string,
    access: AccessLevel,
    clusterId?: string
): Result {
    const defaultResultState = {
        namespaces: [],
        error: '',
        isLoading: true,
    };

    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        if (clusterId !== undefined) {
            getNamespacesForClusterAndPermission({ resource, access }, clusterId)
                .then((data) => {
                    const responseNamespaces = data.namespaces;
                    const rspNamespaces: Namespace[] = [];
                    responseNamespaces.forEach((rspNs: NamespaceForPermission) => {
                        let count = 1;
                        const searchOptions = [
                            { type: 'categoryOption', value: 'Cluster ID:' } as RestSearchOption,
                            { value: clusterId } as RestSearchOption,
                            { type: 'categoryOption', value: 'Namespace:' } as RestSearchOption,
                            { value: rspNs.name } as RestSearchOption,
                        ];
                        fetchDeploymentsCount(searchOptions)
                            .then((value) => {
                                count = value;
                            })
                            .catch(() => {
                                count = 0;
                            });
                        rspNamespaces.push({
                            metadata: { id: rspNs.id, name: rspNs.name },
                            deploymentCount: count,
                        });
                    });
                    setResult({
                        namespaces: rspNamespaces,
                        error: '',
                        isLoading: false,
                    });
                })
                .catch((error) => {
                    const message = getAxiosErrorMessage(error);
                    const errorMessage =
                        message || 'An unknown error occurred while getting the list of clusters';

                    setResult({
                        namespaces: [],
                        error: errorMessage,
                        isLoading: false,
                    });
                });
        }
    }, [access, clusterId, resource]);

    return result;
}

export default useFetchNamespacesForClusterAndPermission;
