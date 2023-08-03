/* eslint-disable no-void */
import { useCallback, useEffect, useState } from 'react';

import {
    fetchReportConfigurations,
    fetchReportStatus,
    fetchReportLastRunStatus,
    fetchReportConfigurationsCount,
} from 'services/ReportsService';

import { ReportStatus } from 'services/ReportsService.types';
import { SearchFilter } from 'types/search';
import { Report } from '../types';
import { getErrorMessage } from '../errorUtils';

export type UseFetchReportsProps = {
    searchFilter: SearchFilter;
    page: number;
    perPage: number;
};

type Result = {
    reports: Report[];
    totalReports: number;
    isLoading: boolean;
    error: string | null;
};

type FetchReportsResult = {
    fetchReports: () => void;
} & Result;

const defaultResult = {
    reports: [],
    totalReports: 0,
    isLoading: false,
    error: null,
};

export function getRequestQueryString(searchFilter: SearchFilter): string {
    return Object.entries(searchFilter)
        .map(([key, val]) => `${key}:${Array.isArray(val) ? val.join(',') : val ?? ''}`)
        .join('+');
}

function useFetchReports({
    searchFilter,
    page,
    perPage,
}: UseFetchReportsProps): FetchReportsResult {
    const [result, setResult] = useState<Result>(defaultResult);

    const fetchReports = useCallback(async () => {
        setResult({
            reports: [],
            totalReports: 0,
            isLoading: true,
            error: null,
        });

        try {
            const reportConfigurations = await fetchReportConfigurations({
                query: getRequestQueryString(searchFilter),
                page,
                perPage,
            });
            const { count: totalReports } = await fetchReportConfigurationsCount({
                query: getRequestQueryString(searchFilter),
                page,
                perPage,
            });
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
                totalReports,
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setResult({
                reports: [],
                totalReports: 0,
                isLoading: false,
                error: getErrorMessage(error),
            });
        }
    }, [searchFilter, page, perPage]);

    useEffect(() => {
        void fetchReports();
    }, [fetchReports, searchFilter, page, perPage]);

    return {
        ...result,
        fetchReports,
    };
}

export default useFetchReports;
