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

const DEPLOYMENTS_FOR_NAMESPACE_QUERY = gql`
    query getNamespaceDeploymentsNames($id: ID!) {
        results: namespace(id: $id) {
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

function useFetchNamespaceDeployments(selectedNamespaceId: string) {
    const [availableNamespaces, setAvailableNamespaces] = useState<string[]>([]);

    // If the selectedNamespaceId has not been set yet, do not run the gql query
    const queryOptions = selectedNamespaceId
        ? { variables: { id: selectedNamespaceId } }
        : { skip: true };

    const { loading, error, data } = useQuery<NamespaceResponse, { id: string }>(
        DEPLOYMENTS_FOR_NAMESPACE_QUERY,
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

export default useFetchNamespaceDeployments;
