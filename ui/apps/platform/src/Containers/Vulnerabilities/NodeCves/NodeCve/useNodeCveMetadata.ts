import { gql, useQuery } from '@apollo/client';

import { ResourceCountsByCveSeverity } from '../../components/BySeveritySummaryCard';
import { CveMetadata } from '../../components/CvePageHeader';

const metadataQuery = gql`
    query getNodeCVEMetadata($cve: String!, $query: String!) {
        totalNodeCount: nodeCount
        nodeCount(query: $query)
        nodeCVE(cve: $cve, subfieldScopeQuery: $query) {
            cve
            distroTuples {
                summary
                link
                operatingSystem
            }
            firstDiscoveredInSystem
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
        }
    }
`;

export default function useNodeCveMetadata(cveId: string, query: string) {
    const metadataRequest = useQuery<
        {
            totalNodeCount: number;
            nodeCount: number;
            nodeCVE: CveMetadata & {
                nodeCountBySeverity: ResourceCountsByCveSeverity;
            };
        },
        {
            cve: string;
            query: string;
        }
    >(metadataQuery, {
        variables: {
            cve: cveId,
            query,
        },
    });

    const { data, previousData } = metadataRequest;
    const nodeCount = data?.nodeCount ?? previousData?.nodeCount ?? 0;
    const cveData = data?.nodeCVE ?? previousData?.nodeCVE;

    return {
        metadataRequest,
        nodeCount,
        cveData,
    };
}
