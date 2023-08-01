import { useCallback, useEffect, useState } from 'react';

import { ReportConfiguration } from 'services/ReportsService.types';
import { fetchReportConfiguration } from 'services/ReportsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

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

    const fetchReportConfig = useCallback(() => {
        setResult(defaultResult);

        fetchReportConfiguration(reportId)
            .then((reportConfiguration) => {
                setResult({
                    reportConfiguration,
                    isLoading: false,
                    error: null,
                });
            })
            .catch((error) => {
                setResult({
                    reportConfiguration: null,
                    isLoading: false,
                    error: getAxiosErrorMessage(error),
                });
            });
    }, [reportId]);

    useEffect(() => {
        fetchReportConfig();
    }, [fetchReportConfig]);

    return result;
}

export default useFetchReport;
