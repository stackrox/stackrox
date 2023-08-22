import { gql, useQuery, ApolloError } from '@apollo/client';

import { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

type DeploymentCountResponse = {
    count: number;
};

type UseFetchDeploymentCount = {
    loading: boolean;
    error: ApolloError | undefined;
    deploymentCount: number | undefined;
};

const DEPLOYMENT_COUNT_QUERY = gql`
    query getDeploymentCount($query: String) {
        count: deploymentCount(query: $query)
    }
`;

function useFetchDeploymentCount(searchFilter: SearchFilter): UseFetchDeploymentCount {
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const queryOptions = { variables: { query } };
    const { loading, error, data } = useQuery<DeploymentCountResponse, { query: string }>(
        DEPLOYMENT_COUNT_QUERY,
        queryOptions
    );

    return {
        loading,
        error,
        deploymentCount: data?.count,
    };
}

export default useFetchDeploymentCount;
