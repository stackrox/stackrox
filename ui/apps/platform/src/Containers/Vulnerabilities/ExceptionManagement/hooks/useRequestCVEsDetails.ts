import { gql, useQuery } from '@apollo/client';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';
// @TODO: Move this up to Containers/Vulnerabilities/utils since it impacts both Workload CVEs and Exception Management.
import { sortCveDistroList } from 'Containers/Vulnerabilities/WorkloadCves/sortUtils';
import { getImageScopeSearchValue } from '../utils';

type AffectedImagesCountQueryResult = {
    imageCVEs: {
        cve: string;
        affectedImageCount: number;
        distroTuples: {
            summary: string;
            operatingSystem: string;
            cvss: number;
            scoreVersion: string;
        }[];
    }[];
};

export const affectedImagesCountQuery = gql`
    query getAffectedImagesCount($query: String) {
        imageCVEs(query: $query) {
            cve
            affectedImageCount
            distroTuples {
                summary
                operatingSystem
                cvss
                scoreVersion
            }
        }
    }
`;

type RequestCVEDetail = {
    cve: string;
    summary: string;
    numAffectedImages: number;
};

type UseAffectedImagesCountResult = {
    isLoading: boolean;
    totalAffectedImageCount: number;
    requestCVEsDetails: RequestCVEDetail[];
};

function useRequestCVEsDetails(exception: VulnerabilityException): UseAffectedImagesCountResult {
    const queryObject = {
        CVE: exception.cves.join(','),
        Image: getImageScopeSearchValue(exception.scope),
    };
    const query = getRequestQueryStringForSearchFilter(queryObject);

    const { loading: isLoading, data } = useQuery<AffectedImagesCountQueryResult>(
        affectedImagesCountQuery,
        {
            variables: {
                query,
            },
        }
    );

    const requestCVEsDetails =
        data?.imageCVEs?.map((imageCVE) => {
            const prioritizedDistros = sortCveDistroList(imageCVE.distroTuples);
            return {
                cve: imageCVE.cve,
                summary: prioritizedDistros.length > 0 ? prioritizedDistros[0].summary : '',
                numAffectedImages: imageCVE.affectedImageCount,
            };
        }) || [];

    const totalAffectedImageCount =
        data?.imageCVEs.reduce((acc, curr) => {
            return acc + curr.affectedImageCount;
        }, 0) || 0;

    return {
        isLoading,
        totalAffectedImageCount,
        requestCVEsDetails,
    };
}

export default useRequestCVEsDetails;
