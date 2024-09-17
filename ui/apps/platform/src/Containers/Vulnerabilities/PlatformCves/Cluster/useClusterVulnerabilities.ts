import { gql, useQuery } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';

import { ClientPagination, Pagination } from 'services/types';
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

export default function useClusterVulnerabilities({
    clusterId,
    query,
    ...pagination
}: { clusterId: string; query: string } & ClientPagination) {
    return useQuery<
        {
            cluster?: {
                clusterVulnerabilityCount: number;
                clusterVulnerabilities: ClusterVulnerability[];
            };
        },
        {
            id: string;
            query: string;
            pagination: Pagination;
        }
    >(clusterVulnerabilitiesQuery, {
        variables: {
            id: clusterId,
            query,
            pagination: getPaginationParams(pagination),
        },
    });
}
