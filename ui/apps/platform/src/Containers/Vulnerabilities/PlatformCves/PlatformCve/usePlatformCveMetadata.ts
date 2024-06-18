import { gql, useQuery } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';

import { ClientPagination } from 'services/types';
import { ClustersByType, clustersByTypeFragment } from './ClustersByTypeSummaryCard';

const platformCveMetadataQuery = gql`
    ${clustersByTypeFragment}
    query getPlatformCVEMetadata($cve: String!, $query: String!) {
        totalClusterCount: clusterCount
        clusterCount(query: $query)
        platformCVE(cve: $cve, subfieldScopeQuery: $query) {
            cve
            distroTuples {
                link
                summary
                operatingSystem
            }
            firstDiscoveredInSystem
            ...ClustersByType
        }
    }
`;

export type PlatformCveMetadata = {
    cve: string;
    distroTuples: {
        link: string;
        summary: string;
        operatingSystem: string;
    }[];
    firstDiscoveredInSystem: string;
};

export default function usePlatformCveMetadata({
    cve,
    query,
    page,
    perPage,
}: { cve: string; query: string } & ClientPagination) {
    return useQuery<{
        totalClusterCount: number;
        clusterCount: number;
        platformCVE: PlatformCveMetadata & ClustersByType;
    }>(platformCveMetadataQuery, {
        variables: {
            cve,
            query,
            pagination: getPaginationParams({ page, perPage }),
        },
    });
}
