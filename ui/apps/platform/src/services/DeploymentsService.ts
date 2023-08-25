import queryString from 'qs';

import searchOptionsToQuery, { RestSearchOption } from 'services/searchOptionsToQuery';
import { Deployment, ListDeployment } from 'types/deployment.proto';
import { ContainerNameAndBaselineStatus } from 'types/processBaseline.proto';
import { Risk } from 'types/risk.proto';
import { ApiSortOption, SearchFilter } from 'types/search';
import {
    ORCHESTRATOR_COMPONENTS_KEY,
    orchestratorComponentsOption,
} from 'utils/orchestratorComponents';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';
import axios from './instance';

const deploymentsUrl = '/v1/deployments';
const deploymentsWithProcessUrl = '/v1/deploymentswithprocessinfo';
const deploymentWithRiskUrl = '/v1/deploymentswithrisk';
const deploymentsCountUrl = '/v1/deploymentscount';

function shouldHideOrchestratorComponents() {
    // for openshift filtering toggle
    return localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true';
}

function fillDeploymentSearchQuery(
    searchFilter: SearchFilter,
    sortOption: Record<string, string> | ApiSortOption,
    page: number,
    pageSize: number
): string {
    const offset = page * pageSize;
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const queryObject: Record<
        string,
        string | Record<string, number | string | Record<string, string> | ApiSortOption>
    > = {
        pagination: {
            offset,
            limit: pageSize,
            sortOption,
        },
    };
    if (query) {
        queryObject.query = query;
    }
    return queryString.stringify(queryObject, { arrayFormat: 'repeat', allowDots: true });
}

/**
 * Fetches list of registered deployments.
 *
 * Changes from the 'fetchDeploymentsLegacy' function:
 * - uses the new `SearchFilter` type instead of `RestSearchOption`
 * - Does not fetch process information linked to the deployment
 */
export function listDeployments(
    searchFilter: SearchFilter,
    sortOption: Record<string, string> | ApiSortOption,
    page: number,
    pageSize: number
): Promise<ListDeployment[]> {
    const params = fillDeploymentSearchQuery(searchFilter, sortOption, page, pageSize);
    return axios
        .get<{ deployments: ListDeployment[] }>(`${deploymentsUrl}?${params}`)
        .then((response) => response?.data?.deployments ?? []);
}

/**
 * Fetches list of registered deployments.
 *
 * Changes from the 'legacy' version of this same function:
 * - returns a 'cancel' function to abort the request
 * - uses the new `SearchFilter` type instead of `RestSearchOption`
 * - Does not implicitly read the value of "shouldHideOrchestratorComponents"
 */
export function fetchDeploymentsWithProcessInfo(
    searchFilter: SearchFilter,
    sortOption: Record<string, string> | ApiSortOption,
    page: number,
    pageSize: number
): CancellableRequest<ListDeploymentWithProcessInfo[]> {
    const params = fillDeploymentSearchQuery(searchFilter, sortOption, page, pageSize);
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ deployments: ListDeploymentWithProcessInfo[] }>(
                `${deploymentsWithProcessUrl}?${params}`,
                {
                    signal,
                }
            )
            .then((response) => response?.data?.deployments ?? [])
    );
}

/**
 * Fetches list of registered deployments.
 */
export function fetchDeploymentsWithProcessInfoLegacy(
    options: RestSearchOption[] = [],
    sortOption: Record<string, string>,
    page: number,
    pageSize: number
): Promise<ListDeploymentWithProcessInfo[]> {
    const offset = page * pageSize;
    let searchOptions: RestSearchOption[] = options;
    if (shouldHideOrchestratorComponents()) {
        searchOptions = [...options, ...orchestratorComponentsOption];
    }
    const query = searchOptionsToQuery(searchOptions);
    const queryObject: Record<
        string,
        string | Record<string, number | string | Record<string, string>>
    > = {
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
        .get<{ deployments: ListDeploymentWithProcessInfo[] }>(
            `${deploymentsWithProcessUrl}?${params}`
        )
        .then((response) => response?.data?.deployments ?? []);
}

export type ListDeploymentWithProcessInfo = {
    deployment: ListDeployment;
    baselineStatuses: ContainerNameAndBaselineStatus[];
};

/**
 * Fetches count of registered deployments.
 */
export function fetchDeploymentsCountLegacy(options: RestSearchOption[]): Promise<number> {
    let searchOptions: RestSearchOption[] = options;
    if (shouldHideOrchestratorComponents()) {
        searchOptions = [...options, ...orchestratorComponentsOption];
    }
    const query = searchOptionsToQuery(searchOptions);
    const queryObject =
        searchOptions.length > 0
            ? {
                  query,
              }
            : {};
    const params = queryString.stringify(queryObject, { arrayFormat: 'repeat' });
    return axios
        .get<{ count: number }>(`${deploymentsCountUrl}?${params}`)
        .then((response) => response?.data?.count ?? 0);
}

export function fetchDeploymentsCount(searchFilter: SearchFilter): Promise<number> {
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const queryObject = query ? { query } : {};
    const params = queryString.stringify(queryObject, { arrayFormat: 'repeat' });
    return axios
        .get<{ count: number }>(`${deploymentsCountUrl}?${params}`)
        .then((response) => response?.data?.count ?? 0);
}

/**
 * Fetches a deployment by its ID.
 */
export function fetchDeployment(id: string): Promise<Deployment> {
    if (!id) {
        throw new Error('Deployment ID must be specified');
    }
    return axios.get<Deployment>(`${deploymentsUrl}/${id}`).then((response) => response.data);
}

/**
 * Fetches a deployment and its risk by deployment ID.
 */
export function fetchDeploymentWithRisk(id: string): Promise<DeploymentWithRisk> {
    if (!id) {
        throw new Error('Deployment ID must be specified');
    }
    return axios
        .get<DeploymentWithRisk>(`${deploymentWithRiskUrl}/${id}`)
        .then((response) => response.data);
}

type DeploymentWithRisk = {
    deployment: Deployment;
    risk: Risk;
};
