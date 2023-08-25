import { useState } from 'react';
import useDeepCompareEffect from 'use-deep-compare-effect';

import forEach from 'lodash/forEach';
import get from 'lodash/get';
import groupBy from 'lodash/groupBy';
import keys from 'lodash/keys';

import { listDeployments } from 'services/DeploymentsService';
import { ListDeployment } from 'types/deployment.proto';
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

function getNamespacesWithDeployments(deployments: ListDeployment[]): NamespaceWithDeployments[] {
    const namespacesWithDeployments: NamespaceWithDeployments[] = [];
    const listDeploymentsIndexedByNamespace = groupBy(deployments, 'namespace');
    const namespaces = keys(listDeploymentsIndexedByNamespace);
    forEach(namespaces, (ns) => {
        const namespaceDeployments: Deployment[] = [];
        const listDeploymentsForNamespace = get(listDeploymentsIndexedByNamespace, ns);
        forEach(listDeploymentsForNamespace, ({ id, name }) => {
            namespaceDeployments.push({ id, name });
        });
        const namespaceWithDeployments: NamespaceWithDeployments = {
            metadata: {
                name: ns,
            },
            deployments: namespaceDeployments,
        };
        namespacesWithDeployments.push(namespaceWithDeployments);
    });
    return namespacesWithDeployments;
}

function useFetchNamespaceDeployments(selectedNamespaceIds: string[]) {
    const [deploymentResponse, setDeploymentResponse] = useState<ListDeploymentResponse>({
        loading: false,
        error: '',
        deploymentsByNamespace: [],
    });
    useDeepCompareEffect(() => {
        if (selectedNamespaceIds.length <= 0) {
            setDeploymentResponse({
                loading: false,
                error: '',
                deploymentsByNamespace: [],
            });
        } else {
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
                    const namespacesWithDeployments: NamespaceWithDeployments[] =
                        getNamespacesWithDeployments(response);
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
    }, [selectedNamespaceIds]);
    return deploymentResponse;
}

export default useFetchNamespaceDeployments;
