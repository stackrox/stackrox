/* eslint-disable no-void */
import { useCallback, useEffect, useState } from 'react';

import { fetchReportHistory } from 'services/ReportsService';
import { ReportSnapshot } from 'services/ReportsService.types';
import { getErrorMessage } from '../errorUtils';

export type UseFetchReportHistory = {
    id: string;
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

function useFetchReportHistory({ id }: UseFetchReportHistory): FetchReportsResult {
    const [result, setResult] = useState<Result>(defaultResult);

    const fetchReportSnapshots = useCallback(async () => {
        setResult({
            reportSnapshots: [],
            isLoading: true,
            error: null,
        });
        try {
            const reportSnapshots = await fetchReportHistory(id);
            setResult({
                reportSnapshots,
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setResult({
                reportSnapshots: [],
                isLoading: false,
                error: getErrorMessage(error),
            });
        }
    }, [id]);

    useEffect(() => {
        void fetchReportSnapshots();
    }, [fetchReportSnapshots]);

    return {
        ...result,
        fetchReportSnapshots,
    };
}

export default useFetchReportHistory;
