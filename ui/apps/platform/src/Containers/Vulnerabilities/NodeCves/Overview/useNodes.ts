import { gql, useQuery } from '@apollo/client';
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
            }
            cluster {
                name
            }
            operatingSystem
            scanTime
        }
    }
`;

// TODO - Verify these types once the BE is implemented
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
    };
    cluster: {
        name: string;
    };
    operatingSystem: string;
    scanTime: string;
};

export default function useNodes(
    querySearchFilter: QuerySearchFilter,
    page: number,
    perPage: number
) {
    return useQuery<{ nodes: Node[] }>(nodeListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
            },
        },
    });
}
