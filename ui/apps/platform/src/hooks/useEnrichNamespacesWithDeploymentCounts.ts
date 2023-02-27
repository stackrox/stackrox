import { useState, useEffect } from 'react';
import { Namespace } from 'hooks/useFetchClusterNamespacesForPermissions';
import { RestSearchOption } from 'services/searchOptionsToQuery';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type EnrichedNamespace = {
    metadata: {
        id: string;
        name: string;
    };
    deploymentCount;
};

export type EnrichedNamespaceResponse = {
    loading: boolean;
    error: string;
    namespaces: EnrichedNamespace[];
};

export function useEnrichNamespacesWithDeploymentCounts(
    namespaceData: Namespace[],
    clusterId?: string
) {
    const [response, setResponse] = useState<EnrichedNamespaceResponse>({
        loading: false,
        error: '',
        namespaces: [],
    });
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
            const enrichedNamespaces: EnrichedNamespace[] = [];
            let shouldBreak = false;
            let errorMessage = 'no error';
            namespaceData.forEach((namespace) => {
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
                            enrichedNamespaces.push({
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
                    namespaces: enrichedNamespaces,
                });
            } else {
                setResponse({
                    loading: false,
                    error: errorMessage,
                    namespaces: [],
                });
            }
        }
    }, [clusterId, namespaceData]);
    return response;
}
