import { gql, useQuery } from '@apollo/client';

import { clustersByTypeFragment } from './ClustersByTypeSummaryCard';
import type { ClustersByType } from './ClustersByTypeSummaryCard';

const platformCveSummaryDataQuery = gql`
    ${clustersByTypeFragment}
    query getPlatformCVEMetadata($cveID: String!, $query: String!) {
        totalClusterCount: clusterCount
        clusterCount(query: $query)
        platformCVE(cveID: $cveID, subfieldScopeQuery: $query) {
            ...ClustersByType
        }
    }
`;

export default function usePlatformCveSummaryData({
    cveId,
    query,
}: {
    cveId: string;
    query: string;
}) {
    return useQuery<{
        totalClusterCount: number;
        clusterCount: number;
        platformCVE?: ClustersByType;
    }>(platformCveSummaryDataQuery, {
        variables: {
            cveID: cveId,
            query,
        },
    });
}
