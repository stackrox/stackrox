/* eslint-disable no-void */
import { useCallback, useEffect, useState } from 'react';

import {
    fetchReportConfigurations,
    fetchReportStatus,
    fetchReportLastRunStatus,
} from 'services/ReportsService';
import { ReportStatus } from 'services/ReportsService.types';
import { Report } from '../types';
import { getErrorMessage } from '../errorUtils';

type Result = {
    reports: Report[];
    isLoading: boolean;
    error: string | null;
};

type FetchReportsResult = {
    fetchReports: () => void;
} & Result;

const defaultResult = {
    reports: [],
    isLoading: false,
    error: null,
};

function useFetchReports(): FetchReportsResult {
    const [result, setResult] = useState<Result>(defaultResult);

    const fetchReports = useCallback(async () => {
        setResult({
            reports: [],
            isLoading: true,
            error: null,
        });

        try {
            const reportConfigurations = await fetchReportConfigurations();
            const reports: Report[] = await Promise.all(
                reportConfigurations.map(async (reportConfiguration): Promise<Report> => {
                    // @TODO: The API returns a 500 when there's no report status. For now we'll do a try/catch, but
                    // we should wait for backend to change this to a 404 or a 200 with a proper message
                    let reportStatus: ReportStatus | null = null;
                    try {
                        reportStatus = await fetchReportStatus(reportConfiguration.id);
                    } catch (error) {
                        reportStatus = null;
                    }
                    const reportLastRunStatus = await fetchReportLastRunStatus(
                        reportConfiguration.id
                    );
                    return {
                        ...reportConfiguration,
                        reportStatus,
                        reportLastRunStatus,
                    };
                })
            );
            setResult({
                reports,
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setResult({
                reports: [],
                isLoading: false,
                error: getErrorMessage(error),
            });
        }
    }, []);

    useEffect(() => {
        void fetchReports();
    }, [fetchReports]);

    return {
        ...result,
        fetchReports,
    };
}

export default useFetchReports;
