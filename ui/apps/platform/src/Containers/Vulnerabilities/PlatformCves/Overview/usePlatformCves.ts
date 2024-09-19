import { gql, useQuery } from '@apollo/client';
import { getPaginationParams } from 'utils/searchUtils';
import { ClientPagination, Pagination } from 'services/types';
import { QuerySearchFilter } from '../../types';
import { getRegexScopedQueryString } from '../../utils/searchUtils';

type PlatformCVE = {
    id: string;
    cve: string;
    isFixable: boolean;
    cveType: string;
    cvss: number;
    clusterVulnerability: {
        scoreVersion: string;
        summary: string;
    };
    clusterCountByType: {
        generic: number;
        kubernetes: number;
        openshift: number;
        openshift4: number;
    };
};

const cveListQuery = gql`
    query getPlatformCves($query: String, $pagination: Pagination) {
        platformCVEs(query: $query, pagination: $pagination) {
            id
            cve
            isFixable
            cveType
            cvss
            clusterVulnerability {
                scoreVersion
                summary
            }
            clusterCountByType {
                generic
                kubernetes
                openshift
                openshift4
            }
        }
    }
`;

export default function usePlatformCves({
    querySearchFilter,
    ...pagination
}: { querySearchFilter: QuerySearchFilter } & ClientPagination) {
    return useQuery<
        {
            platformCVEs: PlatformCVE[];
        },
        {
            query: string;
            pagination: Pagination;
        }
    >(cveListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams(pagination),
        },
    });
}
