import { useState, useEffect } from 'react';
import { gql, useQuery } from '@apollo/client';

type Namespace = {
    metadata: {
        name: string;
    };
};
type NamespaceResponse = {
    id: string;
    results: {
        namespaces: Namespace[];
    };
};

const NAMESPACES_FOR_CLUSTER_QUERY = gql`
    query getClusterNamespaceNames($id: ID!) {
        results: cluster(id: $id) {
            id
            namespaces {
                metadata {
                    name
                }
            }
        }
    }
`;

function useFetchClusterNamespaces(selectedClusterId: string) {
    const [availableNamespaces, setAvailableNamespaces] = useState<string[]>([]);

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

        const namespaces = data.results.namespaces.map(({ metadata }) => metadata.name);

        setAvailableNamespaces(namespaces);
    }, [data]);

    return {
        loading,
        error,
        namespaces: availableNamespaces,
    };
}

export default useFetchClusterNamespaces;
