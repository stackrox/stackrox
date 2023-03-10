import { useEffect, useState } from 'react';

import { useFetchClusterNamespacesForPermissions } from 'hooks/useFetchClusterNamespacesForPermissions';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import { RestSearchOption } from 'services/searchOptionsToQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type NamespaceWithDeploymentCount = {
    metadata: {
        id: string;
        name: string;
    };
    deploymentCount: number;
};

export type NamespacesWithDeploymentCountResponse = {
    loading: boolean;
    error: string;
    namespaces: NamespaceWithDeploymentCount[];
};

export function useFetchClusterNamespacesWithDeploymentCountForPermissions(
    permissions: string[],
    clusterId?: string
): NamespacesWithDeploymentCountResponse {
    const [response, setResponse] = useState<NamespacesWithDeploymentCountResponse>({
        loading: false,
        error: '',
        namespaces: [],
    });

    const { namespaces } = useFetchClusterNamespacesForPermissions(permissions, clusterId);

    useEffect(() => {
        if (clusterId) {
            setResponse({
                loading: true,
                error: '',
                namespaces: [],
            });
            const clusterSearchOptions: RestSearchOption[] = [
                {
                    value: 'Cluster ID:',
                    type: 'categoryOption',
                },
                {
                    value: clusterId,
                },
            ];
            let shouldBreak = false;
            let errorMessage = '';
            const namespacesWithDeploymentCounts: NamespaceWithDeploymentCount[] = [];
            namespaces.forEach((namespace) => {
                if (!shouldBreak) {
                    const namespaceSearchOptions: RestSearchOption[] = [
                        {
                            value: 'Namespace:',
                            type: 'categoryOption',
                        },
                        {
                            value: namespace.name,
                        },
                    ];
                    const searchParameters: RestSearchOption[] = [
                        ...clusterSearchOptions,
                        ...namespaceSearchOptions,
                    ];
                    fetchDeploymentsCount(searchParameters)
                        .then((count) => {
                            namespacesWithDeploymentCounts.push({
                                metadata: {
                                    id: namespace.id,
                                    name: namespace.name,
                                },
                                deploymentCount: count,
                            });
                        })
                        .catch((error) => {
                            const message = getAxiosErrorMessage(error);
                            errorMessage =
                                message ||
                                'An unknown error occurred while getting the number of deployments in namespace';

                            shouldBreak = true;
                        });
                }
            });
            if (!shouldBreak) {
                setResponse({
                    loading: false,
                    error: '',
                    namespaces: namespacesWithDeploymentCounts,
                });
            } else {
                setResponse({
                    loading: false,
                    error: errorMessage,
                    namespaces: [],
                });
            }
        }
    }, [clusterId, namespaces]);

    return response;
}
