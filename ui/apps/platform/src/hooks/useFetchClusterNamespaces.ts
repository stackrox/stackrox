import { useState, useEffect } from 'react';
import { gql, useQuery } from '@apollo/client';

export type Namespace = {
    metadata: {
        id: string;
        name: string;
    };
    deploymentCount;
};
type NamespaceResponse = {
    id: string;
    results: {
        namespaces: Namespace[];
    };
};

const NAMESPACES_FOR_CLUSTER_QUERY = gql`
    query getClusterNamespaces($id: ID!) {
        results: cluster(id: $id) {
            id
            namespaces {
                metadata {
                    id
                    name
                }
                deploymentCount
            }
        }
    }
`;

function useFetchClusterNamespaces(selectedClusterId?: string) {
    const [availableNamespaces, setAvailableNamespaces] = useState<Namespace[]>([]);

    // If the selectedClusterId has not been set yet, do not run the gql query
    const queryOptions = selectedClusterId
        ? { variables: { id: selectedClusterId } }
        : { skip: true };

    const { loading, error, data } = useQuery<NamespaceResponse, { id: string }>(
        NAMESPACES_FOR_CLUSTER_QUERY,
        queryOptions
    );

    useEffect(() => {
        if (!data || !data.results) {
            return;
        }

        setAvailableNamespaces(data.results.namespaces);
    }, [data]);

    return {
        loading,
        error,
        namespaces: availableNamespaces,
    };
}

export default useFetchClusterNamespaces;
