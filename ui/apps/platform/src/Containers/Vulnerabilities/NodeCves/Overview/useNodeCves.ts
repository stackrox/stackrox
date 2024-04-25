import { gql, useQuery } from '@apollo/client';
import { getPaginationParams } from 'utils/searchUtils';
import { QuerySearchFilter } from '../../types';
import { getRegexScopedQueryString } from '../../utils/searchUtils';

const cvesListQuery = gql`
    query getNodeCVEs($query: String, $pagination: Pagination) {
        nodeCVEs(query: $query, pagination: $pagination) {
            cve
            nodeCountBySeverity {
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

// TODO Need to verify these types with the BE implementation
export type NodeCVE = {
    cve: string;
    nodeCountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
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

export default function useNodeCves(
    querySearchFilter: QuerySearchFilter,
    page: number,
    perPage: number
) {
    return useQuery<
        { nodeCVEs: NodeCVE[] },
        {
            query: string;
            pagination: {
                offset: number;
                limit: number;
            };
        }
    >(cvesListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams(page, perPage),
        },
    });
}
