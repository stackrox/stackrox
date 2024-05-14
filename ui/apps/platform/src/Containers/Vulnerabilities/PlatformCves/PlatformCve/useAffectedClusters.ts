import { gql, useQuery } from '@apollo/client';
import { ApiSortOption } from 'types/search';
import { Pagination } from 'services/types';
import { AffectedCluster, affectedClusterFragment } from './AffectedClustersTable';

const affectedClustersQuery = gql`
    ${affectedClusterFragment}
    query getAffectedClusters($query: String, $pagination: Pagination) {
        clusterCount(query: $query)
        clusters(query: $query, pagination: $pagination) {
            ...AffectedClusterFragment
        }
    }
`;

export default function useAffectedClusters(
    query: string,
    page: number,
    perPage: number,
    sortOption: ApiSortOption
) {
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
            pagination: {
                limit: perPage,
                offset: (page - 1) * perPage,
                sortOption,
            },
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
