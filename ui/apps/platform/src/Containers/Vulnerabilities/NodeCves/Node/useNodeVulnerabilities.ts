import { gql, useQuery } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import type { ClientPagination, Pagination } from 'services/types';
import { nodeVulnerabilityFragment } from './CVEsTable';
import type { NodeVulnerability } from './CVEsTable';

const nodeVulnerabilitiesQuery = gql`
    ${nodeVulnerabilityFragment}
    query getNodeVulnerabilities($id: ID!, $query: String!, $pagination: Pagination) {
        node(id: $id) {
            id
            nodeVulnerabilityCount(query: $query)
            nodeVulnerabilities(query: $query, pagination: $pagination) {
                ...NodeVulnerabilityFragment
            }
        }
    }
`;

export default function useNodeVulnerabilities({
    nodeId,
    query,
    ...pagination
}: { nodeId: string; query: string } & ClientPagination) {
    return useQuery<
        {
            node?: {
                nodeVulnerabilityCount: number;
                nodeVulnerabilities: NodeVulnerability[];
            };
        },
        {
            id: string;
            query: string;
            pagination: Pagination;
        }
    >(nodeVulnerabilitiesQuery, {
        variables: {
            id: nodeId,
            query,
            pagination: getPaginationParams(pagination),
        },
    });
}
