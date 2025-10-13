import { useCallback, useEffect, useState } from 'react';

import { fetchReportConfiguration } from 'services/ReportsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ReportConfiguration } from 'services/ReportsService.types';

type FetchReportResult = {
    reportConfiguration: ReportConfiguration | null;
    isLoading: boolean;
    error: string | null;
};

const defaultResult = {
    reportConfiguration: null,
    isLoading: true,
    error: null,
};

function useFetchReport(reportId: string): FetchReportResult {
    const [result, setResult] = useState<FetchReportResult>(defaultResult);

    const fetchReportConfig = useCallback(async () => {
        setResult(defaultResult);

        try {
            const reportConfiguration = await fetchReportConfiguration(reportId);
            setResult({
                reportConfiguration,
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setResult({
                reportConfiguration: null,
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
