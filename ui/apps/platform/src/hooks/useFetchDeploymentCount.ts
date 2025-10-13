import { gql, useQuery } from '@apollo/client';
import type { ApolloError, QueryHookOptions } from '@apollo/client';

import type { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

type DeploymentCountResponse = {
    count: number;
};

type DeploymentCountParameters = {
    query: string;
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

function useFetchDeploymentCount(
    searchFilter: SearchFilter,
    queryOptions: Omit<
        QueryHookOptions<DeploymentCountResponse, DeploymentCountParameters>,
        'variables'
    > = {}
): UseFetchDeploymentCount {
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const options = { ...queryOptions, variables: { query } };
    const { loading, error, data } = useQuery<DeploymentCountResponse, { query: string }>(
        DEPLOYMENT_COUNT_QUERY,
        options
    );

    return {
        loading,
        error,
        deploymentCount: data?.count,
    };
}

export default useFetchDeploymentCount;
