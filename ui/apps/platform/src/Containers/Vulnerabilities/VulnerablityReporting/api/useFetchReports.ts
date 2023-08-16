/* eslint-disable no-void */
import { useCallback, useEffect, useState } from 'react';

import { fetchReportConfigurations, fetchReportConfigurationsCount } from 'services/ReportsService';

import { SearchFilter } from 'types/search';
import { Report } from '../types';
import { getErrorMessage } from '../errorUtils';
import { fetchAndAppendLastReportJobForConfiguration, getRequestQueryString } from './apiUtils';

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
            const { count: totalReports } = await fetchReportConfigurationsCount({
                query: getRequestQueryString(searchFilter),
                page,
                perPage,
            });
            const reports: Report[] = await Promise.all(
                reportConfigurations.map(fetchAndAppendLastReportJobForConfiguration)
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
