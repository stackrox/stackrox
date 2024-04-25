import { gql, useQuery } from '@apollo/client';
import { getPaginationParams } from 'utils/searchUtils';
import { QuerySearchFilter } from '../../types';
import { getRegexScopedQueryString } from '../../utils/searchUtils';

// TODO Validate types with BE implementation
type PlatformCVE = {
    cve: string;
    isFixable: boolean;
    cveType: string;
    cvss: number;
    scoreVersion: string;
    distroTuples: {
        summary: string;
        operatingSystem: string;
        cvss: number;
        scoreVersion: string;
    }[];
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
            cve
            isFixable
            cveType
            cvss
            scoreVersion
            distroTuples {
                summary
                operatingSystem
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

export default function usePlatformCves(
    querySearchFilter: QuerySearchFilter,
    page: number,
    perPage: number
) {
    return useQuery<
        {
            platformCVEs: PlatformCVE[];
        },
        {
            query: string;
            pagination: {
                offset: number;
                limit: number;
            };
        }
    >(cveListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams(page, perPage),
        },
    });
}
