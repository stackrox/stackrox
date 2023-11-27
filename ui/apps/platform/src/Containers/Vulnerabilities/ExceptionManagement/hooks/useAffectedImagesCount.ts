import { gql, useQuery } from '@apollo/client';
import { getImageScope } from '../utils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';

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

function useAffectedImagesCount(exception: VulnerabilityException): UseAffectedImagesCountResult {
    const queryObject = {
        CVE: exception.cves.join(','),
        Image: getImageScope(exception.scope),
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
