import queryString from 'qs';

import {
    isConfiguredReportSnapshot,
    isViewBasedReportSnapshot,
} from 'services/ReportsService.types';
import type {
    ConfiguredReportSnapshot,
    ImageVulnerabilityReportConfiguration,
    ReportHistoryResponse,
    ReportRequestViewBased,
    RunReportResponse,
    RunReportResponseViewBased,
    ViewBasedReportSnapshot,
} from 'services/ReportsService.types';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { buildNestedRawQueryParams, getPaginationParams } from 'utils/searchUtils';
import type { ReportNotificationMethod } from 'types/reportJob';
import { sanitizeFilename } from 'utils/fileUtils';

import axios from './instance';
import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';
import type { Empty } from './types';
import { saveFile } from './DownloadService';

// The following functions are built around the new VM Reporting Enhancements
export const reportDownloadURL = '/api/reports/jobs/download';

// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/report_service.proto

// Configuration of scheduled reports

// Promise because of TS2339 in ImageVulnerabilityReportWizard component
// PostReportConfiguration
export function createReportConfiguration(
    configuration: ImageVulnerabilityReportConfiguration
): Promise<ImageVulnerabilityReportConfiguration> {
    return axios
        .post<ImageVulnerabilityReportConfiguration>('/v2/reports/configurations', configuration)
        .then((response) => response.data);
}

// Promise because of TS2339 in ImageVulnerabilityReportWizard component
// UpdateReportConfiguration
export function updateReportConfiguration(
    reportId: string,
    configuration: ImageVulnerabilityReportConfiguration
): Promise<Empty> {
    return axios
        .put<Empty>(`/v2/reports/configurations/${reportId}`, configuration)
        .then((response) => response.data);
}

// Promise because of @typescript-eslint/await-thenable in useFetchReports hook
// CountReportConfigurations
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
        .then((response) => response.data);
}

// Promise because of @typescript-eslint/await-thenable in useFetchReports hook
// ListReportConfigurations
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
}): Promise<ImageVulnerabilityReportConfiguration[]> {
    const params = queryString.stringify(
        {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios
        .get<{
            reportConfigs: ImageVulnerabilityReportConfiguration[];
        }>(`/v2/reports/configurations?${params}`)
        .then((response) => response?.data?.reportConfigs ?? []);
}

// Promise because of @typescript-eslint/await-thenable in useFetchReport hook
// GetReportConfiguration
export function fetchReportConfiguration(
    reportId: string
): Promise<ImageVulnerabilityReportConfiguration> {
    return axios
        .get<ImageVulnerabilityReportConfiguration>(`/v2/reports/configurations/${reportId}`)
        .then((response) => response.data);
}

// Promise because of @typescript-eslint/await-thenable in useDeleteModal hook
// DeleteReportConfiguration
export function deleteReportConfiguration(reportId: string): Promise<Empty> {
    return axios
        .delete<Empty>(`/v2/reports/configurations/${reportId}`)
        .then((response) => response.data);
}

// Configuration-based jobs

export type FetchReportHistoryServiceParams = {
    id: string;
    query: string;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
    showMyHistory: boolean;
};

// Promise because of @typescript-eslint/await-thenable in useWatchLastSnapshotForReports hook
// GetReportHistory and GetMyReportHistory
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
    const history = showMyHistory ? 'my-history' : 'history';
    return axios
        .get<ReportHistoryResponse>(`/v2/reports/configurations/${id}/${history}?${params}`)
        .then((response) => {
            const snapshots = response.data?.reportSnapshots ?? [];
            return snapshots.filter(isConfiguredReportSnapshot);
        });
}

// Promise because of TS2339 in useRunReport hook
// RunReport
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
        .then((response) => response.data);
}

// Job management

// Promise because of TS2322 in ReportJobs component
// DeleteReport
export function deleteDownloadableReport(reportId: string): Promise<Empty> {
    return axios
        .delete<Empty>(`/v2/reports/jobs/${reportId}/delete`)
        .then((response) => response.data);
}

// View-based jobs

export type FetchViewBasedReportHistoryServiceParams = {
    searchFilter: SearchFilter;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
    showMyHistory: boolean;
};

// GetViewBasedMyReportHistory and GetViewBasedReportHistory
export function fetchViewBasedReportHistory({
    searchFilter,
    page,
    perPage,
    sortOption,
    showMyHistory,
}: FetchViewBasedReportHistoryServiceParams): CancellableRequest<ViewBasedReportSnapshot[]> {
    const params = buildNestedRawQueryParams(
        { searchFilter, page, perPage, sortOption },
        'reportParamQuery'
    );

    const endpoint = showMyHistory
        ? '/v2/reports/view-based/my-history'
        : '/v2/reports/view-based/history';

    return makeCancellableAxiosRequest((signal) =>
        axios.get<ReportHistoryResponse>(`${endpoint}?${params}`, { signal }).then((response) => {
            const snapshots = response.data?.reportSnapshots ?? [];
            return snapshots.filter(isViewBasedReportSnapshot);
        })
    );
}

// PostViewBasedReport
export function runImageViewBasedReport({
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

// Job download

/**
 * Downloads a report file by job ID and saves it to the user's device with a sanitized filename
 * Returns file size in bytes for analytics tracking
 */
export function downloadReportByJobId({
    reportJobId,
    filename,
    fileExtension,
}: {
    reportJobId: string;
    filename: string;
    fileExtension: string;
}): Promise<{ fileSizeBytes?: number }> {
    const sanitizedFilename = sanitizeFilename(filename);

    return saveFile({
        method: 'get',
        url: `/api/reports/jobs/download?id=${reportJobId}`,
        data: null,
        timeout: 300000,
        name: `${sanitizedFilename}.${fileExtension}`,
    });
}
