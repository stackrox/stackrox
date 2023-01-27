import { useState, useEffect } from 'react';
import { gql, useQuery } from '@apollo/client';

import queryService from 'utils/queryService';

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
type DeploymentResponse = {
    results: NamespaceWithDeployments[];
};

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

function useFetchNamespaceDeployments(selectedNamespaceIds: string[]) {
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
    }, [data]);

    return {
        loading,
        error,
        deploymentsByNamespace: availableDeployments,
    };
}

export default useFetchNamespaceDeployments;
