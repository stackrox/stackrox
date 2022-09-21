import queryString from 'qs';

import { Alert, ListAlert } from 'Containers/Violations/types/violationTypes';

import { ApiSortOption, SearchFilter } from 'types/search';
import { getListQueryParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import axios from './instance';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';
import { Empty } from './types';

const baseUrl = '/v1/alerts';
const baseCountUrl = '/v1/alertscount';

// TODO import RestSearchOption and RestSortOption from searchUtils when it is TypeScript.
export type RestSearchOption = {
    label?: string;
    type?: string; // for example, 'categoryOption'
    value: string | string[];
};

// TODO import Severity from PoliciesService when it is TypeScript.
export type Severity = 'LOW_SEVERITY' | 'MEDIUM_SEVERITY' | 'HIGH_SEVERITY' | 'CRITICAL_SEVERITY';

export type AlertEventType = 'CREATED' | 'REMOVED';

export type AlertEvent = {
    time: string; // int64
    type: AlertEventType;
    id: string;
};

export type AlertEventsBySeverity = {
    severity: Severity;
    events: AlertEvent[];
};

export type ClusterAlert = {
    cluster: string;
    severities: AlertEventsBySeverity[];
};

export type AlertCountBySeverity = {
    severity: Severity;
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

/*
 * Fetch a page of list alert objects.
 */
export function fetchAlerts(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page: number,
    pageSize: number
): CancellableRequest<ListAlert[]> {
    const params = getListQueryParams(searchFilter, sortOption, page, pageSize);
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ alerts: ListAlert[] }>(`${baseUrl}?${params}`, { signal })
            .then((response) => response?.data?.alerts ?? [])
    );
}

/*
 * Fetch count of alerts.
 */
export function fetchAlertCount(searchFilter: SearchFilter): CancellableRequest<number> {
    const params = queryString.stringify(
        { query: getRequestQueryStringForSearchFilter(searchFilter) },
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
