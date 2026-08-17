import queryString from 'qs';

import type { ApiSortOption, SearchFilter } from 'types/search';
import { getListQueryParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';
import axios from './instance';
import type { NodeReportConfiguration } from './ReportsService.types';
import type { Empty } from './types';

export function fetchNodeReportConfigurations({
    searchFilter,
    page,
    perPage,
    sortOption,
}: {
    searchFilter: SearchFilter;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
}): CancellableRequest<NodeReportConfiguration[]> {
    const params = getListQueryParams({ searchFilter, sortOption, page, perPage });
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{
                reportConfigs: NodeReportConfiguration[];
            }>(`/v2/reports/node/configurations?${params}`, { signal })
            .then((response) => response?.data?.reportConfigs ?? [])
    );
}

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
            .then((response) => {
                return response.data;
            })
    );
}

export function deleteNodeReportConfiguration(reportId: string): Promise<Empty> {
    return axios
        .delete<Empty>(`/v2/reports/node/configurations/${reportId}`)
        .then((response) => response.data);
}
