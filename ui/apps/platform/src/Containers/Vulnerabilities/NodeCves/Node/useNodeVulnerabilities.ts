import { gql, useQuery } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import { NodeVulnerability, nodeVulnerabilityFragment } from './CVEsTable';

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

export default function useNodeVulnerabilities(
    id: string,
    query: string,
    page: number,
    perPage: number
) {
    return useQuery<
        {
            node: {
                nodeVulnerabilityCount: number;
                nodeVulnerabilities: NodeVulnerability[];
            };
        },
        {
            id: string;
            query: string;
            pagination: { limit: number; offset: number };
        }
    >(nodeVulnerabilitiesQuery, {
        variables: {
            id,
            query,
            pagination: getPaginationParams(page, perPage),
        },
    });
}
