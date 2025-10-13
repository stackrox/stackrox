import { gql, useQuery } from '@apollo/client';
import { getPaginationParams } from 'utils/searchUtils';
import { ClientPagination, Pagination } from 'services/types';
import { QuerySearchFilter } from '../../types';
import { getRegexScopedQueryString } from '../../utils/searchUtils';

const cvesListQuery = gql`
    query getNodeCVEs($query: String, $pagination: Pagination) {
        nodeCVEs(query: $query, pagination: $pagination) {
            cve
            affectedNodeCountBySeverity {
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
            topCVSS
            affectedNodeCount
            firstDiscoveredInSystem
            distroTuples {
                summary
                operatingSystem
                cvss
                scoreVersion
            }
        }
    }
`;

export type NodeCVE = {
    cve: string;
    affectedNodeCountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
        unknown: { total: number };
    };
    topCVSS: number;
    affectedNodeCount: number;
    firstDiscoveredInSystem: string;
    distroTuples: {
        summary: string;
        operatingSystem: string;
        cvss: number;
        scoreVersion: string;
    }[];
};

export default function useNodeCves({
    querySearchFilter,
    ...pagination
}: { querySearchFilter: QuerySearchFilter } & ClientPagination) {
    return useQuery<
        { nodeCVEs: NodeCVE[] },
        {
            query: string;
            pagination: Pagination;
        }
    >(cvesListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams(pagination),
        },
    });
}
