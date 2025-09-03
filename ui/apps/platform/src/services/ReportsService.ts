import queryString from 'qs';

import type {
    ViewBasedReportSnapshot,
    ReportConfiguration,
    ReportHistoryResponse,
    ReportSnapshot,
    RunReportResponse,
    RunReportResponseViewBased,
} from 'services/ReportsService.types';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { getListQueryParams, getPaginationParams } from 'utils/searchUtils';
import type { ReportNotificationMethod, ReportStatus } from 'types/reportJob';
import axios from './instance';
import type { Empty } from './types';

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
}: FetchReportHistoryServiceParams): Promise<ReportSnapshot[]> {
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

export type FetchViewBasedReportHistoryServiceParams = {
    searchFilter: SearchFilter;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
    showMyHistory: boolean;
};

// @TODO: Pass API query information and set up API call to endpoint
export function fetchViewBasedReportHistory({
    searchFilter,
    page,
    perPage,
    sortOption,
    // @TODO: Use the showMyHistory value to determine which endpoint to use
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    showMyHistory,
}: FetchViewBasedReportHistoryServiceParams): Promise<ViewBasedReportSnapshot[]> {
    // @TODO: Use the params in the future API call
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const params = getListQueryParams({ searchFilter, sortOption, page, perPage });

    const mockViewBasedReportJobs: ViewBasedReportSnapshot[] = [
        {
            reportJobId: '3dde30b0-179b-49b4-922d-0d05606c21fb',
            isViewBased: true,
            name: '',
            requestName: 'SC-040925-01',
            areaOfConcern: 'User workloads',
            vulnReportFilters: {
                query: 'Severity:Critical,Important+Image CVE Count:>0',
            },
            reportStatus: {
                runState: 'GENERATED',
                completedAt: '2024-11-13T18:45:32.997367670Z',
                errorMsg: '',
                reportRequestType: 'ON_DEMAND',
                reportNotificationMethod: 'DOWNLOAD',
            },
            user: {
                id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
                name: 'admin',
            },
            isDownloadAvailable: true,
        },
    ];

    return Promise.resolve(mockViewBasedReportJobs);
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
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    query,
    areaOfConcern,
}: {
    query: string;
    areaOfConcern: string;
}): Promise<RunReportResponseViewBased> {
    // TODO: Replace with actual API call when backend is ready
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve({
                reportID: `report-${Date.now()}`,
                requestName: `${areaOfConcern.replace(/\s+/g, '-').toLowerCase()}-${new Date().toISOString().slice(0, 10)}`,
            });
        }, 1500);
    });

    // const requestBody = {
    //     type: 'VULNERABILITY',
    //     viewBasedVulnReportFilters: {
    //         query,
    //     },
    //     areaOfConcern,
    // };

    // return axios
    //     .post<RunReportResponseViewBased>('/v2/reports/view-based/run', requestBody)
    //     .then((response) => response.data);
}
