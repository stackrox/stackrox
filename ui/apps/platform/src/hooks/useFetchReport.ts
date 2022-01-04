import { useEffect, useState } from 'react';

import { fetchReportById } from 'services/ReportsService';
import { ReportConfiguration } from 'types/report.proto';

type Result = { isLoading: boolean; report: ReportConfiguration | null; error: string | null };

const defaultResultState = { report: null, error: null, isLoading: true };

/*
 * This hook does an API call to the report configurations API to get the list of reports
 */
function useFetchReport(reportId: string): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        if (reportId) {
            fetchReportById(reportId)
                .then((data) => {
                    setResult({ report: data || null, error: null, isLoading: false });
                })
                .catch((error) => {
                    setResult({ report: null, error, isLoading: false });
                });
        }
    }, [reportId]);

    return result;
}

export default useFetchReport;
