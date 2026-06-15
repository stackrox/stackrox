import { gql, useQuery } from '@apollo/client';
import { getPaginationParams } from 'utils/searchUtils';
import type { ClientPagination, Pagination } from 'services/types';
import useFeatureFlags from 'hooks/useFeatureFlags';
import type { QuerySearchFilter } from '../../types';
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

const simplifiedCvesListQuery = gql`
    query getNodeCVEsSimplified($query: String, $pagination: Pagination) {
        nodeCVEs(query: $query, pagination: $pagination) {
            cve
            topSeverity
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
    affectedNodeCountBySeverity?: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
        unknown: { total: number };
    };
    topSeverity?: string;
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isSimplifiedSeverity = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_VIEW');

    const query = isSimplifiedSeverity ? simplifiedCvesListQuery : cvesListQuery;

    return useQuery<
        { nodeCVEs: NodeCVE[] },
        {
            query: string;
            pagination: Pagination;
        }
    >(query, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams(pagination),
        },
    });
}
