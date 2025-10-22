import { gql, useQuery } from '@apollo/client';

import { clustersByTypeFragment } from './ClustersByTypeSummaryCard';
import type { ClustersByType } from './ClustersByTypeSummaryCard';

const platformCveMetadataQuery = gql`
    ${clustersByTypeFragment}
    query getPlatformCVEMetadata($cveID: String!) {
        platformCVE(cveID: $cveID) {
            cve
            clusterVulnerability {
                link
                summary
            }
            firstDiscoveredTime
            ...ClustersByType
        }
    }
`;

export type PlatformCveMetadata = {
    cve: string;
    clusterVulnerability: {
        link: string;
        summary: string;
    };
    firstDiscoveredTime: string; // iso8601
};

export default function usePlatformCveMetadata(cveId: string) {
    return useQuery<{
        platformCVE?: PlatformCveMetadata & ClustersByType;
    }>(platformCveMetadataQuery, {
        variables: { cveID: cveId },
    });
}
