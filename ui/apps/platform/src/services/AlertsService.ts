import queryString from 'qs';

import type { Alert, ListAlert } from 'types/alert.proto';
import type { PolicySeverity } from 'types/policy.proto';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { getListQueryParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import axios from './instance';
import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';
import type { Empty } from './types';

const baseUrl = '/v1/alerts';
const baseCountUrl = '/v1/alertscount';

export type AlertCountBySeverity = {
    severity: PolicySeverity;
    count: string;
};

export type AlertGroup = {
    group: string;
    counts: AlertCountBySeverity[];
};

export type AlertQueryGroupBy = 'UNSET' | 'CATEGORY' | 'CLUSTER';

type SummaryAlertCountsFilters = {
    'request.query': string;
    group_by?: AlertQueryGroupBy;
};

/*
 * Fetch severity counts.
 */
export function fetchSummaryAlertCounts(
    filters: SummaryAlertCountsFilters
): CancellableRequest<AlertGroup[]> {
    const params = queryString.stringify(filters);

    // set higher timeout for this call to handle known backend scale issues with dashboard
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ groups: AlertGroup[] }>(`${baseUrl}/summary/counts?${params}`, {
                timeout: 59999,
                signal,
            })
            .then((response) => response.data.groups)
    );
}

type FetchAlertsArguments = {
    alertSearchFilter: SearchFilter;
    sortOption: ApiSortOption;
    page: number;
    perPage: number;
};

/*
 * Fetch a page of list alert objects.
 */
export function fetchAlerts({
    alertSearchFilter,
    sortOption,
    page,
    perPage,
}: FetchAlertsArguments): CancellableRequest<ListAlert[]> {
    const params = getListQueryParams({
        searchFilter: alertSearchFilter,
        sortOption,
        page,
        perPage,
    });
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ alerts: ListAlert[] }>(`${baseUrl}?${params}`, { signal })
            .then((response) => response?.data?.alerts ?? [])
    );
}

/*
 * Fetch count of alerts.
 */
export function fetchAlertCount(alertSearchFilter: SearchFilter): CancellableRequest<number> {
    const params = queryString.stringify(
        { query: getRequestQueryStringForSearchFilter(alertSearchFilter) },
        { arrayFormat: 'repeat' }
    );
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ count: number }>(`${baseCountUrl}?${params}`, { signal })
            .then((response) => response?.data?.count ?? 0)
    );
}

/*
 * Fetch a specified alert.
 */
export function fetchAlert(id: string): Promise<Alert> {
    if (!id) {
        throw new Error('Alert ID must be specified');
    }
    return axios.get<Alert>(`${baseUrl}/${id}`).then((response) => response.data);
}

/*
 * Resolve an alert given an alert ID.
 */
export function resolveAlert(alertId: string, addToBaseline = false): Promise<Empty> {
    return axios
        .patch<Empty>(`${baseUrl}/${alertId}/resolve`, { addToBaseline })
        .then((response) => response.data);
}

/*
 * Resolve a list of alerts by alert ID.
 */
export function resolveAlerts(alertIds: string[] = [], addToBaseline = false): Promise<Empty[]> {
    return Promise.all(alertIds.map((id) => resolveAlert(id, addToBaseline)));
}
