import { gql, useQuery } from '@apollo/client';

const totalNodeCountQuery = gql`
    query getTotalNodeCount {
        nodeCount
    }
`;

export default function useTotalNodeCount() {
    const totalNodeCountRequest = useQuery<{ nodeCount: number }>(totalNodeCountQuery);

    return totalNodeCountRequest.data?.nodeCount ?? 0;
}
