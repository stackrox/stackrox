import { useState, useEffect } from 'react';

import { listDeployments } from 'services/DeploymentsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type Deployment = {
    id: string;
    name: string;
};
export type NamespaceWithDeployments = {
    metadata: {
        name: string;
    };
    deployments: Deployment[];
};

type ListDeploymentResponse = {
    loading: boolean;
    error: string;
    deploymentsByNamespace: NamespaceWithDeployments[];
};

function useFetchNamespaceDeployments(selectedNamespaceIds: string[]) {
    const [deploymentResponse, setDeploymentResponse] = useState<ListDeploymentResponse>({
        loading: false,
        error: '',
        deploymentsByNamespace: [],
    });
    const selectedNamespaceIdsAsString = JSON.stringify(selectedNamespaceIds);
    useEffect(() => {
        const namespaceIds = JSON.parse(selectedNamespaceIdsAsString);
        if (namespaceIds.length > 0) {
            setDeploymentResponse({
                loading: true,
                error: '',
                deploymentsByNamespace: [],
            });
            const searchQuery: Record<string, string[]> = {
                'Namespace ID': namespaceIds,
            };
            const sortOption = { field: 'Deployment', reversed: 'false' };
            listDeployments(searchQuery, sortOption, 0, 0)
                .then((response) => {
                    const namespacesWithDeployments: NamespaceWithDeployments[] = [];
                    const deploymentsByNamespace = new Map<string, Deployment[]>();
                    response.forEach(({ id, name, namespace }) => {
                        const deployment = { id, name };
                        const deploymentList = deploymentsByNamespace.get(namespace);
                        if (deploymentList) {
                            deploymentList.push(deployment);
                            deploymentsByNamespace.set(namespace, deploymentList);
                        } else {
                            deploymentsByNamespace.set(namespace, [deployment]);
                        }
                    });
                    deploymentsByNamespace.forEach((deployments, namespaceName) => {
                        const namespaceWithDeployments: NamespaceWithDeployments = {
                            metadata: {
                                name: namespaceName,
                            },
                            deployments,
                        };
                        namespacesWithDeployments.push(namespaceWithDeployments);
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
    }, [selectedNamespaceIdsAsString]);
    return deploymentResponse;
}

export default useFetchNamespaceDeployments;
