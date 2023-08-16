/* eslint-disable no-void */
import { useCallback, useEffect, useState } from 'react';

import {
    fetchReportConfigurations,
    fetchReportConfigurationsCount,
    fetchReportHistory,
} from 'services/ReportsService';

import { SearchFilter } from 'types/search';
import { Report } from '../types';
import { getErrorMessage } from '../errorUtils';
import { getRequestQueryString } from './apiUtils';

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
            const { count: totalReports } = await fetchReportConfigurationsCount(
                getRequestQueryString(searchFilter)
            );
            const reports: Report[] = await Promise.all(
                reportConfigurations.map(async (reportConfiguration): Promise<Report> => {
                    const PAGE = 1;
                    const PER_PAGE = 1;
                    const SHOW_MY_HISTORY = true;
                    // Query for the current user's last report job
                    const query = getRequestQueryString({
                        'Report state': ['PREPARING', 'WAITING'],
                    });
                    const reportSnapshot = await fetchReportHistory(
                        reportConfiguration.id,
                        query,
                        PAGE,
                        PER_PAGE,
                        SHOW_MY_HISTORY
                    );
                    return {
                        ...reportConfiguration,
                        reportSnapshot: reportSnapshot?.[0] || null,
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
