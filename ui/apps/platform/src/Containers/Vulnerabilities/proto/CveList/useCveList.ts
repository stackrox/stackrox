import { gql, useQuery } from '@apollo/client';

export type ProtoCVEListItem = {
    cveName: string;
    severity: string;
    cvss: number;
    imageCount: number;
    fixable: boolean;
    firstSeen: string;
};

export const PROTO_CVE_LIST = gql`
    query protoCVEList($limit: Int, $offset: Int) {
        protoCVEList(limit: $limit, offset: $offset) {
            cveName
            severity
            cvss
            imageCount
            fixable
            firstSeen
        }
    }
`;

type ProtoCVEListData = {
    protoCVEList: ProtoCVEListItem[];
};

type ProtoCVEListVars = {
    limit?: number;
    offset?: number;
};

/**
 * Fetches the prototype CVE list from the GraphQL API.
 */
export function useCveList(limit = 50, offset = 0) {
    return useQuery<ProtoCVEListData, ProtoCVEListVars>(PROTO_CVE_LIST, {
        variables: { limit, offset },
        fetchPolicy: 'cache-and-network',
    });
}
