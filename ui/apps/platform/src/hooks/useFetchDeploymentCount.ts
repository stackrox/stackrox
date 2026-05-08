import { gql, useQuery } from '@apollo/client';
import type { ApolloError, QueryHookOptions } from '@apollo/client';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { SearchFilter } from 'types/search';
import { withActiveDeploymentQuery } from 'utils/deploymentUtils';
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isDeploymentSoftDeletionEnabled = isFeatureFlagEnabled('ROX_DEPLOYMENT_SOFT_DELETION');

    const query = withActiveDeploymentQuery(
        getRequestQueryStringForSearchFilter(searchFilter),
        isDeploymentSoftDeletionEnabled
    );
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
