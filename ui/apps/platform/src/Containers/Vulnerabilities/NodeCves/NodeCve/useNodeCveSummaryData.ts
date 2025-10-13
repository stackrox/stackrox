import { gql, useQuery } from '@apollo/client';

import { ResourceCountsByCveSeverity } from '../../components/BySeveritySummaryCard';

const summaryDataQuery = gql`
    query getNodeCVESummaryData($cve: String!, $query: String!) {
        totalNodeCount: nodeCount
        nodeCount(query: $query)
        nodeCVE(cve: $cve, subfieldScopeQuery: $query) {
            distroTuples {
                operatingSystem
            }
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
        }
    }
`;

export default function useNodeCveSummaryData(cveId: string, query: string) {
    const summaryDataRequest = useQuery<
        {
            totalNodeCount: number;
            nodeCount: number;
            nodeCVE?: {
                distroTuples: {
                    operatingSystem: string;
                }[];
                affectedNodeCountBySeverity: ResourceCountsByCveSeverity;
            };
        },
        {
            cve: string;
            query: string;
        }
    >(summaryDataQuery, {
        variables: {
            cve: cveId,
            query,
        },
    });

    const { data, previousData } = summaryDataRequest;
    const nodeCount = data?.nodeCount ?? previousData?.nodeCount ?? 0;

    return {
        summaryDataRequest,
        nodeCount,
    };
}
