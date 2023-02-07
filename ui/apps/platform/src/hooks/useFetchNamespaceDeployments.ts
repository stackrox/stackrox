import { useState, useEffect } from 'react';
// import { gql, useQuery } from '@apollo/client';

import { listDeployments } from 'services/DeploymentsService';
// import queryService from 'utils/queryService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type Deployment = {
    id: string;
    name: string;
};
export type NamespaceWithDeployments = {
    metadata: {
        id: string;
        name: string;
    };
    deployments: Deployment[];
};
/*
type DeploymentResponse = {
    results: NamespaceWithDeployments[];
};

 */
type ListDeploymentResponse = {
    loading: boolean;
    error: string;
    deploymentsByNamespace: NamespaceWithDeployments[];
};

/*
const DEPLOYMENTS_FOR_NAMESPACE_QUERY = gql`
    query getNamespaceDeployments($query: String!) {
        results: namespaces(query: $query) {
            metadata {
                name
                id
            }
            deployments {
                name
                id
            }
        }
    }
`;

 */

function useFetchNamespaceDeployments(
    selectedNamespaceIds: string[],
    namesToIdMapping: Map<string, string>
) {
    const [deploymentResponse, setDeploymentResponse] = useState<ListDeploymentResponse>({
        loading: false,
        error: '',
        deploymentsByNamespace: [],
    });
    useEffect(() => {
        if (selectedNamespaceIds.length > 0) {
            setDeploymentResponse({
                loading: true,
                error: '',
                deploymentsByNamespace: [],
            });
            const searchQuery: Record<string, string[]> = {
                'Namespace ID': selectedNamespaceIds,
            };
            const sortOption = { field: 'Deployment', reversed: 'false' };
            listDeployments(searchQuery, sortOption, 0, 0)
                .then((response) => {
                    const namespacesWithDeployments: NamespaceWithDeployments[] = [];
                    const deploymentsByNamespace = new Map<string, Deployment[]>();
                    response.forEach((listDeployment) => {
                        const deployment: Deployment = {
                            id: listDeployment.id,
                            name: listDeployment.name,
                        };
                        const { namespace } = listDeployment;
                        if (!deploymentsByNamespace.has(namespace)) {
                            deploymentsByNamespace.set(namespace, []);
                        }
                        deploymentsByNamespace[namespace].push(deployment);
                    });
                    deploymentsByNamespace.forEach((deployments, namespace) => {
                        const namespaceId = namesToIdMapping.get(namespace);
                        if (namespaceId) {
                            const namespaceWithDeployments: NamespaceWithDeployments = {
                                metadata: {
                                    id: namespaceId,
                                    name: namespace,
                                },
                                deployments,
                            };
                            namespacesWithDeployments.push(namespaceWithDeployments);
                        }
                    });
                    setDeploymentResponse({
                        loading: false,
                        error: '',
                        deploymentsByNamespace: namespacesWithDeployments,
                    });
                })
                .catch((error) => {
                    const message = getAxiosErrorMessage(error);
                    const errorMessage =
                        message ||
                        'An unknown error occurred while getting the list of deployments';

                    setDeploymentResponse({
                        loading: false,
                        error: errorMessage,
                        deploymentsByNamespace: [],
                    });
                });
        }
    }, [namesToIdMapping, selectedNamespaceIds]);
    return deploymentResponse;
    /*
    const [availableDeployments, setAvailableNamespaces] = useState<NamespaceWithDeployments[]>([]);

    const searchClause = { 'Namespace ID': selectedNamespaceIds };
    // If the selectedNamespaceId has not been set yet, do not run the gql query
    const queryOptions =
        selectedNamespaceIds.length > 0
            ? { variables: { query: queryService.objectToWhereClause(searchClause) } }
            : { skip: true };

    const { loading, error, data } = useQuery<DeploymentResponse, { query: string }>(
        DEPLOYMENTS_FOR_NAMESPACE_QUERY,
        queryOptions
    );

    useEffect(() => {
        if (!data || !data.results) {
            setAvailableNamespaces([]);
        } else {
            setAvailableNamespaces(data.results);
        }
        // clean up state on unmount
        return () => setAvailableNamespaces([]);
    }, [data]);

    return {
        loading,
        error,
        deploymentsByNamespace: availableDeployments,
    };

     */
}

export default useFetchNamespaceDeployments;
