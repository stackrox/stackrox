import queryString from 'qs';

import {
    ReportConfiguration,
    ReportHistoryResponse,
    ReportSnapshot,
    RunReportResponse,
} from 'services/ReportsService.types';
import { ApiSortOption } from 'types/search';
import { getPaginationParams } from 'utils/searchUtils';
import { ReportNotificationMethod, ReportStatus } from 'types/reportJob';
import axios from './instance';
import { Empty } from './types';

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

export type FetchReportHistoryServiceProps = {
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
}: FetchReportHistoryServiceProps): Promise<ReportSnapshot[]> {
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
            return response.data?.reportSnapshots ?? [];
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
