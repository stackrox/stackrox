import { gql, useQuery } from '@apollo/client';
import type { ClientPagination, Pagination } from 'services/types';
import { getPaginationParams } from 'utils/searchUtils';
import { affectedNodeFragment } from './AffectedNodesTable';
import type { AffectedNode } from './AffectedNodesTable';

const affectedNodesQuery = gql`
    ${affectedNodeFragment}
    query getAffectedNodes($query: String, $pagination: Pagination) {
        nodes(query: $query, pagination: $pagination) {
            ...AffectedNode
        }
    }
`;

export default function useAffectedNodes({
    query,
    ...pagination
}: { query: string } & ClientPagination) {
    const affectedNodesRequest = useQuery<
        {
            nodes: AffectedNode[];
        },
        {
            query: string;
            pagination: Pagination;
        }
    >(affectedNodesQuery, {
        variables: {
            query,
            pagination: getPaginationParams(pagination),
        },
    });

    return {
        affectedNodesRequest,
        nodeData: affectedNodesRequest.data?.nodes ?? affectedNodesRequest.previousData?.nodes,
    };
}
