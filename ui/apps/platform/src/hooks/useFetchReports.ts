import { useState } from 'react';
import useDeepCompareEffect from 'use-deep-compare-effect';

import { fetchReports, fetchReportsCount } from 'services/ReportsService';
import { RestSearchOption } from 'services/searchOptionsToQuery';
import { ReportConfiguration } from 'types/report.proto';
import { ApiSortOption } from 'types/search';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { convertToRestSearch } from 'utils/searchUtils';

type Result = {
    isLoading: boolean;
    reports: ReportConfiguration[] | null;
    reportCount: number;
    error: string | null;
    triggerRefresh: () => void;
};

/*
 * This hook does an API call to the report configurations API to get the list of reports
 */
function useFetchReports(
    filteredSearch: Record<string, string>,
    sortOption: ApiSortOption,
    currentPage: number,
    perPage: number
): Result {
    const defaultResultState = {
        reports: null,
        reportCount: 0,
        error: null,
        isLoading: true,
        triggerRefresh,
    };

    const [result, setResult] = useState<Result>(defaultResultState);
    const [refetchFlag, setRefetchFlag] = useState(0);

    const restSearch: RestSearchOption[] = convertToRestSearch(filteredSearch || {});

    useDeepCompareEffect(() => {
        setResult(defaultResultState);

        Promise.all<[Promise<ReportConfiguration[] | null>, Promise<number>]>([
            fetchReports(restSearch || [], sortOption, currentPage - 1, perPage),
            fetchReportsCount(restSearch || []),
        ])
            .then((data) => {
                const reportsResponse = data[0];
                const countResponse = data[1];
                setResult({
                    reports: reportsResponse || null,
                    reportCount: countResponse,
                    error: null,
                    isLoading: false,
                    triggerRefresh,
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of reports';

                setResult({
                    reports: null,
                    reportCount: 0,
                    error: errorMessage,
                    isLoading: false,
                    triggerRefresh,
                });
            });
    }, [refetchFlag, restSearch, sortOption, currentPage, perPage]);

    function triggerRefresh() {
        setRefetchFlag((flag) => flag + 1);
    }

    return result;
}

export default useFetchReports;
