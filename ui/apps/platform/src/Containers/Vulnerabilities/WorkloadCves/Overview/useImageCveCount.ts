import { gql, useQuery } from '@apollo/client';

/**
 * GAP: there is no REST endpoint for imageCVECount.
 * /v1/nodecves and /v1/clustercves only expose suppress/unsuppress — no list/count for image CVEs.
 * Keep this hook on Apollo until a typed REST count exists.
 */
const imageCveCountQuery = gql`
    query getImageCVECount($query: String) {
        imageCVECount(query: $query)
    }
`;

export default function useImageCveCount(query: string) {
    return useQuery<{ imageCVECount: number }>(imageCveCountQuery, {
        variables: { query },
    });
}
