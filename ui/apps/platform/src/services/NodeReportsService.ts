import queryString from 'qs';

import type { ReportNotificationMethod } from 'types/reportJob';
import type { ApiSortOption, SearchFilter, SearchQueryOptions } from 'types/search';
import {
    buildNestedRawQueryParams,
    getListQueryParams,
    getPaginationParams,
    getRequestQueryStringForSearchFilter,
} from 'utils/searchUtils';

import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';
import axios from './instance';
import { isConfiguredReportSnapshot } from './ReportsService.types';
import type {
    ConfiguredReportSnapshot,
    NodeViewBasedReportSnapshot,
    NodeVulnerabilityReportConfiguration,
    ReportRequestViewBased,
    RunReportResponse,
    RunReportResponseViewBased,
} from './ReportsService.types';
import type { Empty } from './types';

// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/node_report_service.proto

// Configuration of scheduled reports

// PostNodeReportConfiguration
export function createNodeReportConfiguration(
    configuration: NodeVulnerabilityReportConfiguration
): Promise<NodeVulnerabilityReportConfiguration> {
    return axios
        .post<NodeVulnerabilityReportConfiguration>(
            '/v2/reports/node/configurations',
            configuration
        )
        .then((response) => response.data);
}

// UpdateNodeReportConfiguration
export function updateNodeReportConfiguration(
    reportId: string,
    configuration: NodeVulnerabilityReportConfiguration
): Promise<Empty> {
    return axios
        .put<Empty>(`/v2/reports/node/configurations/${reportId}`, configuration)
        .then((response) => response.data);
}

// ListNodeReportConfigurations
export function fetchNodeReportConfigurations({
    searchFilter,
    page,
    perPage,
    sortOption,
}: SearchQueryOptions): CancellableRequest<NodeVulnerabilityReportConfiguration[]> {
    const params = getListQueryParams({ searchFilter, sortOption, page, perPage });
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{
                reportConfigs: NodeVulnerabilityReportConfiguration[];
            }>(`/v2/reports/node/configurations?${params}`, { signal })
            .then((response) => response?.data?.reportConfigs ?? [])
    );
}

// CountNodeReportConfigurations
export function fetchNodeReportConfigurationsCount(
    searchFilter: SearchFilter
): CancellableRequest<{ count: number }> {
    const params = queryString.stringify(
        { query: getRequestQueryStringForSearchFilter(searchFilter) },
        { arrayFormat: 'repeat' }
    );
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ count: number }>(`/v2/reports/node/configuration-count?${params}`, { signal })
            .then((response) => response.data)
    );
}

// GetNodeReportConfiguration
export function fetchNodeReportConfiguration(
    reportId: string
): Promise<NodeVulnerabilityReportConfiguration> {
    return axios
        .get<NodeVulnerabilityReportConfiguration>(`/v2/reports/node/configurations/${reportId}`)
        .then((response) => response.data);
}

// DeleteNodeReportConfiguration
export function deleteNodeReportConfiguration(reportId: string): Promise<Empty> {
    return axios
        .delete<Empty>(`/v2/reports/node/configurations/${reportId}`)
        .then((response) => response.data);
}

// Configuration-based jobs

// RunNodeReport
export function runNodeReportRequest(
    reportConfigId: string,
    reportNotificationMethod: ReportNotificationMethod
): Promise<RunReportResponse> {
    return axios
        .post<RunReportResponse>('/v2/reports/node/run', {
            reportConfigId,
            reportNotificationMethod,
        })
        .then((response) => response.data);
}

// GetNodeReportHistory / GetMyNodeReportHistory
export function fetchNodeReportHistory({
    id,
    query,
    page,
    perPage,
    sortOption,
    showMyHistory,
}: {
    id: string;
    query: string;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
    showMyHistory: boolean;
}): Promise<ConfiguredReportSnapshot[]> {
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
        .get<{
            reportSnapshots: ConfiguredReportSnapshot[];
        }>(`/v2/reports/node/configurations/${id}/${history}?${params}`)
        .then((response) => {
            const snapshots = response.data?.reportSnapshots ?? [];
            return snapshots.filter(isConfiguredReportSnapshot);
        });
}

// Job management

// DeleteNodeReport
export function deleteNodeDownloadableReport(reportId: string): Promise<Empty> {
    return axios
        .delete<Empty>(`/v2/reports/node/jobs/${reportId}/delete`)
        .then((response) => response.data);
}

export const nodeReportDownloadURL = '/api/reports/node/jobs/download';

// View-based jobs

// PostViewBasedNodeReport
export function runNodeViewBasedReport({
    query,
    areaOfConcern,
}: {
    query: string;
    areaOfConcern: string;
}): Promise<RunReportResponseViewBased> {
    const requestBody: ReportRequestViewBased = {
        type: 'NODE_VULNERABILITY',
        nodeVulnReportFilters: {
            query,
        },
        areaOfConcern, // 'Nodes' for analytics ony
    };

    return axios
        .post<RunReportResponseViewBased>('/v2/reports/node/view-based/run', requestBody)
        .then((response) => response.data);
}

// GetViewBasedNodeReportHistory
export function getViewBasedNodeReportHistory({
    searchFilter,
    page,
    perPage,
    sortOption,
}: SearchQueryOptions): Promise<NodeViewBasedReportSnapshot[]> {
    const params = buildNestedRawQueryParams(
        { searchFilter, page, perPage, sortOption },
        'reportParamQuery'
    );

    return axios
        .get<{
            reportSnapshots: NodeViewBasedReportSnapshot[];
        }>(`/v2/reports/node/view-based/history?${params}`)
        .then((response) => response.data?.reportSnapshots ?? []);
}

// GetViewBasedMyNodeReportHistory
export function getViewBasedMyNodeReportHistory({
    searchFilter,
    page,
    perPage,
    sortOption,
}: SearchQueryOptions): Promise<NodeViewBasedReportSnapshot[]> {
    const params = buildNestedRawQueryParams(
        { searchFilter, page, perPage, sortOption },
        'reportParamQuery'
    );

    return axios
        .get<{
            reportSnapshots: NodeViewBasedReportSnapshot[];
        }>(`/v2/reports/node/view-based/my-history?${params}`)
        .then((response) => response.data?.reportSnapshots ?? []);
}
