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
    // Query for the current user's last report job
    const query = getRequestQueryString({ 'Report state': ['PREPARING', 'WAITING'] });

    const reportSnapshot = await fetchReportHistory({
        id: reportConfiguration.id,
        query,
        page: 1,
        perPage: 1,
        showMyHistory: true,
        sortOption: {
            field: 'Report Completion Time',
            reversed: true,
        },
    });
    return {
        ...reportConfiguration,
        reportSnapshot: reportSnapshot[0] ?? null,
    };
}
