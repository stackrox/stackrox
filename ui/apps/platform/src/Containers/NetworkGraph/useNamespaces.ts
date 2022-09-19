import { useState, useEffect } from 'react';
import { gql, useQuery } from '@apollo/client';

type NamespaceMetadataResp = {
    id: string;
    results: {
        namespaces: {
            metadata: {
                name: string;
            };
        }[];
    };
};

export const NAMESPACES_FOR_CLUSTER_QUERY = gql`
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

function useNamespaces(selectedClusterId) {
    const [availableNamespace, setAvailableNamespace] = useState<string[]>([]);
    // If the selectedClusterId has not been set yet, do not run the gql query
    const queryOptions = selectedClusterId
        ? { variables: { id: selectedClusterId } }
        : { skip: true };

    const { loading, error, data } = useQuery<NamespaceMetadataResp, { id: string }>(
        NAMESPACES_FOR_CLUSTER_QUERY,
        queryOptions
    );

    useEffect(() => {
        if (!data || !data.results) {
            return;
        }

        const namespaces = data.results.namespaces.map(({ metadata }) => metadata.name);

        setAvailableNamespace(namespaces);
    }, [data]);

    return {
        loading,
        error,
        namespaces: availableNamespace,
    };
}

export default useNamespaces;
