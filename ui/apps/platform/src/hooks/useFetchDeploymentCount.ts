import { useState, useEffect } from 'react';
import { gql, useQuery, ApolloError } from '@apollo/client';

type DeploymentCountResponse = {
    count: number;
};

type UseFetchDeploymentCount = {
    loading: boolean;
    error: ApolloError | undefined;
    deploymentCount: number | undefined;
};

const DEPLOYMENT_COUNT_FOR_CLUSTER = gql`
    query deployments($query: String) {
        count: deploymentCount(query: $query)
    }
`;

function useFetchDeploymentCount(selectedClusterId: string): UseFetchDeploymentCount {
    const [deploymentCount, setDeploymentCount] = useState<number>();

    // If the selectedClusterId has not been set yet, do not run the gql query
    const queryOptions = selectedClusterId
        ? { variables: { id: selectedClusterId } }
        : { skip: true };

    const { loading, error, data } = useQuery<DeploymentCountResponse, { id: string }>(
        DEPLOYMENT_COUNT_FOR_CLUSTER,
        queryOptions
    );

    useEffect(() => {
        if (!data || !data.count) {
            return;
        }

        setDeploymentCount(data.count);
    }, [data]);

    return {
        loading,
        error,
        deploymentCount,
    };
}

export default useFetchDeploymentCount;
