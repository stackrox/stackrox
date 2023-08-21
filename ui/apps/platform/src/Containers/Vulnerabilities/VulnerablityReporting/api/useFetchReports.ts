/* eslint-disable no-void */
import { useCallback, useEffect, useState } from 'react';

import { fetchReportConfigurations, fetchReportConfigurationsCount } from 'services/ReportsService';

import { ApiSortOption, SearchFilter } from 'types/search';
import { ReportConfiguration } from 'services/ReportsService.types';
import { getErrorMessage } from '../errorUtils';
import { getRequestQueryString } from './apiUtils';

export type UseFetchReportsProps = {
    searchFilter: SearchFilter;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
};

type Result = {
    reportConfigurations: ReportConfiguration[] | null;
    totalReports: number;
    isLoading: boolean;
    error: string | null;
};

type FetchReportsResult = {
    fetchReports: () => void;
} & Result;

const defaultResult = {
    reportConfigurations: null,
    totalReports: 0,
    isLoading: false,
    error: null,
};

function useFetchReports({
    searchFilter,
    page,
    perPage,
    sortOption,
}: UseFetchReportsProps): FetchReportsResult {
    const [result, setResult] = useState<Result>(defaultResult);

    const fetchReports = useCallback(async () => {
        setResult((prevResult) => ({
            reportConfigurations: prevResult.reportConfigurations,
            totalReports: prevResult.totalReports,
            isLoading: true,
            error: null,
        }));

        try {
            const reportConfigurations = await fetchReportConfigurations({
                query: getRequestQueryString(searchFilter),
                page,
                perPage,
                sortOption,
            });
            const { count: totalReports } = await fetchReportConfigurationsCount({
                query: getRequestQueryString(searchFilter),
            });
            setResult({
                reportConfigurations,
                totalReports,
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setResult({
                reportConfigurations: null,
                totalReports: 0,
                isLoading: false,
                error: getErrorMessage(error),
            });
        }
    }, [searchFilter, page, perPage, sortOption]);

    useEffect(() => {
        void fetchReports();
    }, [fetchReports, searchFilter, page, perPage]);

    return {
        ...result,
        fetchReports,
    };
}

export default useFetchReports;
