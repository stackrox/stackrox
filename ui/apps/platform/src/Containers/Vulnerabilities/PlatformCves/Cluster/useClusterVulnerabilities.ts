import { gql, useQuery } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';

import { ClusterVulnerability, clusterVulnerabilityFragment } from './CVEsTable';

const clusterVulnerabilitiesQuery = gql`
    ${clusterVulnerabilityFragment}
    query getClusterVulnerabilities($id: ID!, $query: String!, $pagination: Pagination) {
        cluster(id: $id) {
            id
            clusterVulnerabilityCount(query: $query)
            clusterVulnerabilities(query: $query, pagination: $pagination) {
                ...ClusterVulnerabilityFragment
            }
        }
    }
`;

export default function useClusterVulnerabilities(
    id: string,
    query: string,
    page: number,
    perPage: number
) {
    return useQuery<
        {
            cluster: {
                clusterVulnerabilityCount: number;
                clusterVulnerabilities: ClusterVulnerability[];
            };
        },
        {
            id: string;
            query: string;
            pagination: { limit: number; offset: number };
        }
    >(clusterVulnerabilitiesQuery, {
        variables: {
            id,
            query,
            pagination: getPaginationParams(page, perPage),
        },
    });
}
