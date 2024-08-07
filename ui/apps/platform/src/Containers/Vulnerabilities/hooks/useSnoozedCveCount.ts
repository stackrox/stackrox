import { gql, useQuery } from '@apollo/client';

const snoozedNodeCveCountQuery = gql`
    query getSnoozedNodeCveCount {
        count: nodeCVECount(query: "CVE Snoozed:true")
    }
`;

const snoozedPlatformCveCountQuery = gql`
    query getSnoozedPlatformCveCount {
        count: platformCVECount(query: "CVE Snoozed:true")
    }
`;

export default function useSnoozedCveCount(cveType: 'Node' | 'Platform'): number | undefined {
    const totalNodeCountRequest = useQuery<{ count: number }>(
        cveType === 'Node' ? snoozedNodeCveCountQuery : snoozedPlatformCveCountQuery
    );

    return totalNodeCountRequest.data?.count;
}
