import { gql, useQuery } from '@apollo/client';
import { getPaginationParams } from 'utils/searchUtils';
import { ClientPagination } from 'services/types';
import { getRegexScopedQueryString } from '../../utils/searchUtils';
import { QuerySearchFilter } from '../../types';

const nodeListQuery = gql`
    query getNodes($query: String, $pagination: Pagination) {
        nodes(query: $query, pagination: $pagination) {
            id
            name
            nodeCVECountBySeverity {
                critical {
                    total
                }
                important {
                    total
                }
                moderate {
                    total
                }
                low {
                    total
                }
                unknown {
                    total
                }
            }
            cluster {
                name
            }
            osImage
            scanTime
        }
    }
`;

type Node = {
    id: string;
    name: string;
    nodeCVECountBySeverity: {
        critical: {
            total: number;
        };
        important: {
            total: number;
        };
        moderate: {
            total: number;
        };
        low: {
            total: number;
        };
        unknown: {
            total: number;
        };
    };
    cluster: {
        name: string;
    };
    osImage: string;
    scanTime: string;
};

export default function useNodes({
    querySearchFilter,
    ...pagination
}: { querySearchFilter: QuerySearchFilter } & ClientPagination) {
    return useQuery<{ nodes: Node[] }>(nodeListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams(pagination),
        },
    });
}
