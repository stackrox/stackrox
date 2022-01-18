import { useState } from 'react';
import useDeepCompareEffect from 'use-deep-compare-effect';

import { fetchReports, fetchReportsCount } from 'services/ReportsService';
import { RestSortOption } from 'services/sortOption';
import { RestSearchOption } from 'services/searchOptionsToQuery';
import { ReportConfiguration } from 'types/report.proto';
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
function useFetchReport(
    filteredSearch: Record<string, string>,
    sortOption: RestSortOption,
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
    const [refetchFlag, setRefetchFlag] = useState<number>(new Date().getTime());

    const restSearch: RestSearchOption[] = convertToRestSearch(filteredSearch || {});

    useDeepCompareEffect(() => {
        setResult(defaultResultState);

        Promise.all<[Promise<ReportConfiguration[] | null>, Promise<number>]>([
            fetchReports(restSearch || [], sortOption, currentPage - 1, perPage),
            fetchReportsCount(),
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
        setRefetchFlag(new Date().getTime());
    }

    return result;
}

export default useFetchReport;
