import { useCallback, useEffect, useState } from 'react';

import { fetchReportConfiguration } from 'services/ReportsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { fetchAndAppendLastReportJobForConfiguration } from './apiUtils';
import { Report } from '../types';

type FetchReportResult = {
    report: Report | null;
    isLoading: boolean;
    error: string | null;
};

const defaultResult = {
    report: null,
    isLoading: true,
    error: null,
};

function useFetchReport(reportId: string): FetchReportResult {
    const [result, setResult] = useState<FetchReportResult>(defaultResult);

    const fetchReportConfig = useCallback(async () => {
        setResult(defaultResult);

        try {
            const reportConfiguration = await fetchReportConfiguration(reportId);
            const report = await fetchAndAppendLastReportJobForConfiguration(reportConfiguration);
            setResult({
                report,
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setResult({
                report: null,
                isLoading: false,
                error: getAxiosErrorMessage(error),
            });
        }
    }, [reportId]);

    useEffect(() => {
        // eslint-disable-next-line @typescript-eslint/no-floating-promises
        fetchReportConfig();
    }, [fetchReportConfig]);

    return result;
}

export default useFetchReport;
