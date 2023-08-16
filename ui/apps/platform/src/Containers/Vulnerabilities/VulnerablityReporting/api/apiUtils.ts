import { fetchReportHistory } from 'services/ReportsService';
import { ReportConfiguration } from 'services/ReportsService.types';
import { SearchFilter } from 'types/search';

import { Report } from '../types';

export function getRequestQueryString(searchFilter: SearchFilter): string {
    return Object.entries(searchFilter)
        .map(([key, val]) => `${key}:${Array.isArray(val) ? val.join(',') : val ?? ''}`)
        .join('+');
}

export async function fetchAndAppendLastReportJobForConfiguration(
    reportConfiguration: ReportConfiguration
): Promise<Report> {
    const PAGE = 1;
    const PER_PAGE = 1;
    const SHOW_MY_HISTORY = true;
    // Query for the current user's last report job
    const query = getRequestQueryString({ 'Report state': ['PREPARING', 'WAITING'] });

    const reportSnapshot = await fetchReportHistory(
        reportConfiguration.id,
        query,
        PAGE,
        PER_PAGE,
        SHOW_MY_HISTORY
    );
    return {
        ...reportConfiguration,
        reportSnapshot: reportSnapshot[0] ?? null,
    };
}
