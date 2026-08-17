import queryString from 'qs';

import type { SearchFilter, SearchQueryOptions } from 'types/search';
import { getListQueryParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';
import type { NodeVulnerabilityReportConfiguration } from './ReportsService.types';
import axios from './instance';

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
