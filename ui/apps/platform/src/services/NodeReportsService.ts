import queryString from 'qs';

import type { SearchFilter, SearchQueryOptions } from 'types/search';
import {
    buildNestedRawQueryParams,
    getListQueryParams,
    getRequestQueryStringForSearchFilter,
} from 'utils/searchUtils';

import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';
import axios from './instance';
import type {
    NodeViewBasedReportSnapshot,
    NodeVulnerabilityReportConfiguration,
} from './ReportsService.types';
import type { Empty } from './types';

// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/node_report_service.proto

// Configuration of scheduled reports

// PostNodeReportConfiguration
export function createNodeReportConfiguration(
    configuration: NodeVulnerabilityReportConfiguration
): CancellableRequest<NodeVulnerabilityReportConfiguration> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .post<NodeVulnerabilityReportConfiguration>(
                '/v2/reports/node/configurations',
                configuration,
                { signal }
            )
            .then((response) => response.data)
    );
}

// UpdateNodeReportConfiguration
export function updateNodeReportConfiguration(
    reportId: string,
    configuration: NodeVulnerabilityReportConfiguration
): CancellableRequest<Empty> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .put<Empty>(`/v2/reports/node/configurations/${reportId}`, configuration, { signal })
            .then((response) => response.data)
    );
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
): CancellableRequest<NodeVulnerabilityReportConfiguration> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<NodeVulnerabilityReportConfiguration>(
                `/v2/reports/node/configurations/${reportId}`,
                { signal }
            )
            .then((response) => response.data)
    );
}

// DeleteNodeReportConfiguration
export function deleteNodeReportConfiguration(reportId: string): Promise<Empty> {
    return axios
        .delete<Empty>(`/v2/reports/node/configurations/${reportId}`)
        .then((response) => response.data);
}

// Configuration-based jobs

// RunNodeReport

// GetNodeReportHistory

// GetMyNodeReportHistory

// Job management

// GetNodeReportStatus

// CancelNodeReport

// DeleteNodeReport

// View-based jobs

// PostViewBasedNodeReport

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
