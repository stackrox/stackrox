import { gql, useQuery } from '@apollo/client';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';
import { getImageScopeSearchValue } from '../utils';

type AffectedImagesCountQueryResult = {
    imageCVEs: {
        cve: string;
        affectedImageCount: number;
    }[];
};

type UseAffectedImagesCountResult = {
    isAffectedImagesCountLoading: boolean;
    affectedImagesCount: number;
};

export const affectedImagesCountQuery = gql`
    query getAffectedImagesCount($query: String) {
        imageCVEs(query: $query) {
            cve
            affectedImageCount
        }
    }
`;

// @TODO: Refactor this hook for ROX-20555 (reference: https://github.com/stackrox/stackrox/pull/8798#discussion_r1409315044)
function useAffectedImagesCount(exception: VulnerabilityException): UseAffectedImagesCountResult {
    const queryObject = {
        CVE: exception.cves.join(','),
        Image: getImageScopeSearchValue(exception.scope),
    };
    const query = getRequestQueryStringForSearchFilter(queryObject);

    const { loading: isAffectedImagesCountLoading, data } =
        useQuery<AffectedImagesCountQueryResult>(affectedImagesCountQuery, {
            variables: {
                query,
            },
        });

    const affectedImagesCount =
        data?.imageCVEs.reduce((acc, curr) => {
            return acc + curr.affectedImageCount;
        }, 0) || 0;

    return { isAffectedImagesCountLoading, affectedImagesCount };
}

export default useAffectedImagesCount;
