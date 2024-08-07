import { gql, useQuery } from '@apollo/client';
import {
    PlatformCVECountByStatus,
    platformCveCountByStatusFragment,
} from './PlatformCvesByStatusSummaryCard';
import {
    PlatformCVECountByType,
    platformCveCountByTypeFragment,
} from './PlatformCvesByTypeSummaryCard';

export const clusterSummaryDataQuery = gql`
    ${platformCveCountByStatusFragment}
    ${platformCveCountByTypeFragment}
    query getClusterVulnSummary($id: ID!, $query: String) {
        cluster(id: $id) {
            id
            platformCVECountByFixability(query: $query) {
                ...PlatformCveCountByStatusFragment
            }
            platformCVECountByType(query: $query) {
                ...PlatformCveCountByTypeFragment
            }
        }
    }
`;

export default function useClusterSummaryData(clusterId: string, query: string) {
    return useQuery<
        {
            cluster: {
                id: string;
                platformCVECountByFixability: PlatformCVECountByStatus;
                platformCVECountByType: PlatformCVECountByType;
            };
        },
        { id: string; query: string }
    >(clusterSummaryDataQuery, {
        variables: { id: clusterId, query },
    });
}
