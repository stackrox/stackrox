import { gql, useQuery } from '@apollo/client';
import { getPaginationParams } from 'utils/searchUtils';
import type { ClientPagination } from 'services/types';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { getRegexScopedQueryString } from '../../utils/searchUtils';
import type { QuerySearchFilter } from '../../types';

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

const simplifiedNodeListQuery = gql`
    query getNodesSimplified($query: String, $pagination: Pagination) {
        nodes(query: $query, pagination: $pagination) {
            id
            name
            topCvss
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
    nodeCVECountBySeverity?: {
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
    topCvss?: number;
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isSimplifiedSeverity = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_VIEW');

    const query = isSimplifiedSeverity ? simplifiedNodeListQuery : nodeListQuery;

    return useQuery<{ nodes: Node[] }>(query, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams(pagination),
        },
    });
}
