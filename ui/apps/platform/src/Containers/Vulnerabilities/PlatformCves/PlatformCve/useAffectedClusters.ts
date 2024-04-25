import { gql, useQuery } from '@apollo/client';
import { AffectedCluster, affectedClusterFragment } from './AffectedClustersTable';

const affectedClustersQuery = gql`
    ${affectedClusterFragment}
    query getAffectedClusters($query: String, $pagination: Pagination) {
        clusters(query: $query, pagination: $pagination) {
            ...AffectedClusterFragment
        }
    }
`;

export default function useAffectedClusters(query: string, page: number, perPage: number) {
    const affectedClustersRequest = useQuery<
        {
            clusters: AffectedCluster[];
        },
        {
            query: string;
            pagination: { limit: number; offset: number };
        }
    >(affectedClustersQuery, {
        variables: {
            query,
            pagination: {
                limit: perPage,
                offset: (page - 1) * perPage,
            },
        },
    });

    return {
        affectedClustersRequest,
        clusterData:
            affectedClustersRequest.data?.clusters ??
            affectedClustersRequest.previousData?.clusters,
    };
}
