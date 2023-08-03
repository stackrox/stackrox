import queryString from 'qs';

import { ReportConfiguration as ReportConfigurationV1 } from 'types/report.proto';
import {
    ReportConfiguration,
    ReportHistoryResponse,
    ReportSnapshot,
    ReportNotificationMethod,
    ReportStatus,
    RunReportResponse,
} from 'services/ReportsService.types';
import searchOptionsToQuery, { RestSearchOption } from 'services/searchOptionsToQuery';
import { ApiSortOption } from 'types/search';
import axios from './instance';
import { Empty } from './types';

const reportUrl = '/v1/report';
const reportServiceUrl = `${reportUrl}/run`;
const reportConfigurationsUrl = `${reportUrl}/configurations`;
const reportConfigurationsCountUrl = '/v1/report-configurations-count';

export function fetchReports(
    options: RestSearchOption[] = [],
    sortOption: ApiSortOption,
    page: number,
    pageSize: number
): Promise<ReportConfigurationV1[]> {
    const offset = page * pageSize;
    const searchOptions: RestSearchOption[] = [...options];
    const query = searchOptionsToQuery(searchOptions);
    const queryObject: Record<string, string | Record<string, number | string | ApiSortOption>> = {
        pagination: {
            offset,
            limit: pageSize,
            sortOption,
        },
    };
    if (query) {
        queryObject.query = query;
    }
    const params = queryString.stringify(queryObject, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<{ reportConfigs: ReportConfigurationV1[] }>(`${reportConfigurationsUrl}?${params}`)
        .then((response) => {
            return response?.data?.reportConfigs ?? [];
        });
}

// TODO: need a way to get total reports count properly
export function fetchReportsCount(options: RestSearchOption[] = []): Promise<number> {
    const searchOptions: RestSearchOption[] = [...options];
    const query = searchOptionsToQuery(searchOptions);
    const queryObject =
        searchOptions.length > 0
            ? {
                  query,
              }
            : {};

    const params = queryString.stringify(queryObject, { arrayFormat: 'repeat', allowDots: true });

    return axios
        .get<{ count: number }>(`${reportConfigurationsCountUrl}?${params}`)
        .then((response) => {
            return response?.data?.count ?? 0;
        });
}

export function fetchReportById(reportId: string): Promise<ReportConfigurationV1> {
    return axios
        .get<{ reportConfig: ReportConfigurationV1 }>(`${reportConfigurationsUrl}/${reportId}`)
        .then((response) => {
            return response?.data?.reportConfig ?? {};
        });
}

export function saveReport(report: ReportConfigurationV1): Promise<ReportConfigurationV1> {
    const apiPayload = {
        reportConfig: report,
    };

    const promise = report.id
        ? axios.put<ReportConfigurationV1>(`${reportConfigurationsUrl}/${report.id}`, apiPayload)
        : axios.post<ReportConfigurationV1>(reportConfigurationsUrl, apiPayload);

    return promise.then((response) => {
        return response.data;
    });
}

export function deleteReport(reportId: string): Promise<Empty> {
    return axios.delete(`${reportConfigurationsUrl}/${reportId}`);
}

export function runReport(reportId: string): Promise<Empty> {
    return axios.post(`${reportServiceUrl}/${reportId}`);
}

// The following functions are built around the new VM Reporting Enhancements

// @TODO: Same logic is used in fetchReportConfigurations. Maybe consider something more DRY
export function fetchReportConfigurationsCount({
    query,
    page,
    perPage,
}: {
    query: string;
    page: number;
    perPage: number;
}): Promise<{ count: number }> {
    const params = queryString.stringify(
        {
            query,
            pagination: {
                limit: perPage,
                offset: page - 1,
            },
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
}: {
    query: string;
    page: number;
    perPage: number;
}): Promise<ReportConfiguration[]> {
    const params = queryString.stringify(
        {
            query,
            pagination: {
                limit: perPage,
                offset: page - 1,
            },
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
        .get<{ status: ReportStatus | null }>(`/v2/reports/status/${id}`)
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

export function fetchReportHistory(id: string): Promise<ReportSnapshot[]> {
    return axios
        .get<ReportHistoryResponse>(`/v2/reports/configurations/${id}/history`)
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
