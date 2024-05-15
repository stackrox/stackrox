import { gql, useQuery } from '@apollo/client';

import { QuerySearchFilter } from '../../types';
import { getRegexScopedQueryString } from '../../utils/searchUtils';

const entityCountsQuery = gql`
    query getNodeCVEEntityCounts($query: String) {
        nodeCVECount(query: $query)
        nodeCount(query: $query)
    }
`;

export function useNodeCveEntityCounts(querySearchFilter: QuerySearchFilter) {
    return useQuery<
        {
            nodeCVECount: number;
            nodeCount: number;
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
