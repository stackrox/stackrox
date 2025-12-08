import { gql, useQuery } from '@apollo/client';
import type { ClientPagination, Pagination } from 'services/types';
import { getPaginationParams } from 'utils/searchUtils';
import { affectedClusterFragment } from './AffectedClustersTable';
import type { AffectedCluster } from './AffectedClustersTable';

const affectedClustersQuery = gql`
    ${affectedClusterFragment}
    query getAffectedClusters($query: String, $pagination: Pagination) {
        clusterCount(query: $query)
        clusters(query: $query, pagination: $pagination) {
            ...AffectedClusterFragment
        }
    }
`;

export default function useAffectedClusters({
    query,
    ...pagination
}: { query: string } & ClientPagination) {
    const affectedClustersRequest = useQuery<
        {
            clusterCount: number;
            clusters: AffectedCluster[];
        },
        {
            query: string;
            pagination: Pagination;
        }
    >(affectedClustersQuery, {
        variables: {
            query,
            pagination: getPaginationParams(pagination),
        },
    });

    return {
        affectedClustersRequest,
        clusterCount: affectedClustersRequest.data?.clusterCount ?? 0,
        clusterData:
            affectedClustersRequest.data?.clusters ??
            affectedClustersRequest.previousData?.clusters,
    };
}
