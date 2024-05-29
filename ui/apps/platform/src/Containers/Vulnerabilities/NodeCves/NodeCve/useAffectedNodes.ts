import { gql, useQuery } from '@apollo/client';
import { ApiSortOption } from 'types/search';
import { Pagination } from 'services/types';
import { getPaginationParams } from 'utils/searchUtils';
import { affectedNodeFragment, AffectedNode } from './AffectedNodesTable';

const affectedNodesQuery = gql`
    ${affectedNodeFragment}
    query getAffectedNodes($query: String, $pagination: Pagination) {
        nodes(query: $query, pagination: $pagination) {
            ...AffectedNode
        }
    }
`;

export default function useAffectedNodes(
    query: string,
    page: number,
    perPage: number,
    sortOption: ApiSortOption
) {
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
            pagination: { ...getPaginationParams(page, perPage), sortOption },
        },
    });

    return {
        affectedNodesRequest,
        nodeData: affectedNodesRequest.data?.nodes ?? affectedNodesRequest.previousData?.nodes,
    };
}
