import { gql, useQuery } from '@apollo/client';

import { QuerySearchFilter } from '../../types';
import { getRegexScopedQueryString } from '../../utils/searchUtils';

const entityCountsQuery = gql`
    query getPlatformCVEEntityCounts($query: String) {
        platformCVECount(query: $query)
        clusterCount(query: $query)
    }
`;

export function usePlatformCveEntityCounts(querySearchFilter: QuerySearchFilter) {
    return useQuery<
        {
            platformCVECount: number;
            clusterCount: number;
        },
        {
            query: string;
        }
    >(entityCountsQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
        },
    });
}
