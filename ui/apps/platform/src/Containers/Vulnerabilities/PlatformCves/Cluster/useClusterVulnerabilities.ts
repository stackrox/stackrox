import { gql, useQuery } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';

import { Pagination } from 'services/types';
import { ApiSortOption } from 'types/search';
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
    perPage: number,
    sortOption: ApiSortOption
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
            pagination: Pagination;
        }
    >(clusterVulnerabilitiesQuery, {
        variables: {
            id,
            query,
            pagination: { ...getPaginationParams(page, perPage), sortOption },
        },
    });
}
