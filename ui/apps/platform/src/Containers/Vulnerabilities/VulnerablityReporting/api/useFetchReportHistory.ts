/* eslint-disable no-void */
import { useCallback, useEffect, useState } from 'react';

import { fetchReportHistory } from 'services/ReportsService';
import { ReportSnapshot } from 'services/ReportsService.types';
import { ApiSortOption } from 'types/search';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type UseFetchReportHistory = {
    id: string;
    query: string;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
    showMyHistory: boolean;
};

type Result = {
    reportSnapshots: ReportSnapshot[];
    isLoading: boolean;
    error: string | null;
};

export type FetchReportsResult = {
    fetchReportSnapshots: () => void;
} & Result;

const defaultResult = {
    reportSnapshots: [],
    isLoading: false,
    error: null,
};

function useFetchReportHistory({
    id,
    query,
    page,
    perPage,
    sortOption,
    showMyHistory,
}: UseFetchReportHistory): FetchReportsResult {
    const [result, setResult] = useState<Result>(defaultResult);

    const fetchReportSnapshots = useCallback(() => {
        setResult({
            reportSnapshots: [],
            isLoading: true,
            error: null,
        });
        fetchReportHistory({ id, query, page, perPage, sortOption, showMyHistory })
            .then((reportSnapshots) => {
                setResult({
                    reportSnapshots,
                    isLoading: false,
                    error: null,
                });
            })
            .catch((error) => {
                setResult({
                    reportSnapshots: [],
                    isLoading: false,
                    error: getAxiosErrorMessage(error),
                });
            });
    }, [id, query, page, perPage, sortOption, showMyHistory]);

    useEffect(() => {
        void fetchReportSnapshots();
    }, [fetchReportSnapshots]);

    return {
        ...result,
        fetchReportSnapshots,
    };
}

export default useFetchReportHistory;
