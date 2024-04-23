import { gql, useQuery } from '@apollo/client';
import { affectedNodeFragment, AffectedNode } from './AffectedNodesTable';

const affectedNodesQuery = gql`
    ${affectedNodeFragment}
    query getAffectedNodes($query: String, $pagination: Pagination) {
        nodes(query: $query, pagination: $pagination) {
            ...AffectedNode
        }
    }
`;

export default function useAffectedNodes(query: string, page: number, perPage: number) {
    const affectedNodesRequest = useQuery<
        {
            nodes: AffectedNode[];
        },
        {
            query: string;
            pagination: { limit: number; offset: number };
        }
    >(affectedNodesQuery, {
        variables: {
            query,
            pagination: {
                limit: perPage,
                offset: (page - 1) * perPage,
            },
        },
    });

    return {
        affectedNodesRequest,
        nodeData: affectedNodesRequest.data?.nodes ?? affectedNodesRequest.previousData?.nodes,
    };
}
