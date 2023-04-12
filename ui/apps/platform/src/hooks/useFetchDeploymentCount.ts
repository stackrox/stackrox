import { gql, useQuery, ApolloError } from '@apollo/client';

type DeploymentCountResponse = {
    cluster: {
        count: number;
    };
};

type UseFetchDeploymentCount = {
    loading: boolean;
    error: ApolloError | undefined;
    deploymentCount: number | undefined;
};

const DEPLOYMENT_COUNT_FOR_CLUSTER = gql`
    query getDeploymentCountForCluster($id: ID!) {
        cluster(id: $id) {
            count: deploymentCount
        }
    }
`;

function useFetchDeploymentCount(selectedClusterId: string): UseFetchDeploymentCount {
    // If the selectedClusterId has not been set yet, do not run the gql query
    const queryOptions = selectedClusterId
        ? { variables: { id: selectedClusterId } }
        : { skip: true };

    const { loading, error, data } = useQuery<DeploymentCountResponse, { id: string }>(
        DEPLOYMENT_COUNT_FOR_CLUSTER,
        queryOptions
    );

    return {
        loading,
        error,
        deploymentCount: data?.cluster?.count || undefined,
    };
}

export default useFetchDeploymentCount;
