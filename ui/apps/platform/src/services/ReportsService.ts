import queryString from 'qs';

import {
    isConfiguredReportSnapshot,
    isViewBasedReportSnapshot,
} from 'services/ReportsService.types';
import type {
    ConfiguredReportSnapshot,
    ReportConfiguration,
    ReportHistoryResponse,
    ReportRequestViewBased,
    RunReportResponse,
    RunReportResponseViewBased,
    ViewBasedReportSnapshot,
} from 'services/ReportsService.types';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { getListQueryParams, getPaginationParams } from 'utils/searchUtils';
import type { ReportNotificationMethod, ReportStatus } from 'types/reportJob';
import { sanitizeFilename } from 'utils/fileUtils';

import axios from './instance';
import type { Empty } from './types';
import { saveFile } from './DownloadService';

// The following functions are built around the new VM Reporting Enhancements
export const reportDownloadURL = '/api/reports/jobs/download';

// @TODO: Same logic is used in fetchReportConfigurations. Maybe consider something more DRY
export function fetchReportConfigurationsCount({
    query,
}: {
    query: string;
}): Promise<{ count: number }> {
    const params = queryString.stringify(
        {
            query,
        },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios
        .get<{ count: number }>(`/v2/reports/configuration-count?${params}`)
        .then((response) => {
            return response.data;
        });
}

export function fetchReportConfigurations({
    query,
    page,
    perPage,
    sortOption,
}: {
    query: string;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
}): Promise<ReportConfiguration[]> {
    const params = queryString.stringify(
        {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios
        .get<{ reportConfigs: ReportConfiguration[] }>(`/v2/reports/configurations?${params}`)
        .then((response) => {
            return response?.data?.reportConfigs ?? [];
        });
}

export function fetchReportConfiguration(reportId: string): Promise<ReportConfiguration> {
    return axios
        .get<ReportConfiguration>(`/v2/reports/configurations/${reportId}`)
        .then((response) => {
            return response.data;
        });
}

export function fetchReportStatus(id: string): Promise<ReportStatus | null> {
    return axios
        .get<{ status: ReportStatus | null }>(`/v2/reports/jobs/${id}/status`)
        .then((response) => {
            return response.data?.status;
        });
}

export function fetchReportLastRunStatus(id: string): Promise<ReportStatus | null> {
    return axios
        .get<{ status: ReportStatus | null }>(`/v2/reports/last-status/${id}`)
        .then((response) => {
            return response.data?.status;
        });
}

export type FetchReportHistoryServiceParams = {
    id: string;
    query: string;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
    showMyHistory: boolean;
};

export function fetchReportHistory({
    id,
    query,
    page,
    perPage,
    sortOption,
    showMyHistory,
}: FetchReportHistoryServiceParams): Promise<ConfiguredReportSnapshot[]> {
    const params = queryString.stringify(
        {
            reportParamQuery: {
                query,
                pagination: getPaginationParams({ page, perPage, sortOption }),
            },
        },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios
        .get<ReportHistoryResponse>(
            `/v2/reports/configurations/${id}/${showMyHistory ? 'my-history' : 'history'}?${params}`
        )
        .then((response) => {
            const snapshots = response.data?.reportSnapshots ?? [];
            return snapshots.filter(isConfiguredReportSnapshot);
        });
}

export type FetchViewBasedReportHistoryServiceParams = {
    searchFilter: SearchFilter;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
    showMyHistory: boolean;
};

export function fetchViewBasedReportHistory({
    searchFilter,
    page,
    perPage,
    sortOption,
    showMyHistory,
}: FetchViewBasedReportHistoryServiceParams): Promise<ViewBasedReportSnapshot[]> {
    const params = getListQueryParams({ searchFilter, sortOption, page, perPage });

    const endpoint = showMyHistory
        ? '/v2/reports/view-based/my-history'
        : '/v2/reports/view-based/history';

    return axios.get<ReportHistoryResponse>(`${endpoint}?${params}`).then((response) => {
        const snapshots = response.data?.reportSnapshots ?? [];
        return snapshots.filter(isViewBasedReportSnapshot);
    });
}

export function createReportConfiguration(
    report: ReportConfiguration
): Promise<ReportConfiguration> {
    return axios
        .post<ReportConfiguration>('/v2/reports/configurations', report)
        .then((response) => {
            return response.data;
        });
}

export function updateReportConfiguration(
    reportId: string,
    report: ReportConfiguration
): Promise<ReportConfiguration> {
    return axios
        .put<ReportConfiguration>(`/v2/reports/configurations/${reportId}`, report)
        .then((response) => {
            return response.data;
        });
}

export function deleteReportConfiguration(reportId: string): Promise<Empty> {
    return axios.delete<Empty>(`/v2/reports/configurations/${reportId}`).then((response) => {
        return response.data;
    });
}

// @TODO: Rename this to runReport when we remove the old report code
export function runReportRequest(
    reportConfigId: string,
    reportNotificationMethod: ReportNotificationMethod
): Promise<RunReportResponse> {
    return axios
        .post<RunReportResponse>('/v2/reports/run', {
            reportConfigId,
            reportNotificationMethod,
        })
        .then((response) => {
            return response.data;
        });
}

export function downloadReport(reportId: string) {
    return axios.get<string>(`/v2/reports/jobs/${reportId}/download`).then((response) => {
        return response.data;
    });
}

export function deleteDownloadableReport(reportId: string) {
    return axios.delete<Empty>(`/v2/reports/jobs/${reportId}/delete`).then((response) => {
        return response.data;
    });
}

export function runViewBasedReport({
    query,
    areaOfConcern,
}: {
    query: string;
    areaOfConcern: string;
}): Promise<RunReportResponseViewBased> {
    const requestBody: ReportRequestViewBased = {
        type: 'VULNERABILITY',
        viewBasedVulnReportFilters: {
            query,
        },
        areaOfConcern,
    };

    return axios
        .post<RunReportResponseViewBased>('/v2/reports/view-based/run', requestBody)
        .then((response) => response.data);
}

/**
 * Downloads a report file by job ID and saves it to the user's device with a sanitized filename
 */
export function downloadReportByJobId({
    reportJobId,
    filename,
    fileExtension,
}: {
    reportJobId: string;
    filename: string;
    fileExtension: string;
}): Promise<void> {
    const sanitizedFilename = sanitizeFilename(filename);

    return saveFile({
        method: 'get',
        url: `/api/reports/jobs/download?id=${reportJobId}`,
        data: null,
        timeout: 300000,
        name: `${sanitizedFilename}.${fileExtension}`,
    });
}
