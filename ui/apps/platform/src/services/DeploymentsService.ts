import queryString from 'qs';

import type { Deployment, ListDeployment } from 'types/deployment.proto';
import type { ContainerNameAndBaselineStatus } from 'types/processBaseline.proto';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';
import axios from './instance';
import type { Pagination } from './types';

const deploymentsUrl = '/v1/deployments';
const deploymentsWithProcessUrl = '/v1/deploymentswithprocessinfo';
const deploymentWithRiskUrl = '/v1/deploymentswithrisk';
const deploymentsCountUrl = '/v1/deploymentscount';

export type Risk = {
    id: string;
    subject: RiskSubject;
    score: number; // float
    results: RiskResult[];
};

export type RiskResult = {
    name: string;
    factors: RiskFactor[];
    score: number; // float
};

export type RiskFactor = {
    message: string;
    url: string;
};

export type RiskSubject = {
    id: string;
    namespace: string;
    clusterId: string;
    type: RiskSubjectType;
};

export type RiskSubjectType =
    | 'UNKNOWN'
    | 'DEPLOYMENT'
    | 'NAMESPACE'
    | 'CLUSTER'
    | 'NODE'
    | 'NODE_COMPONENT'
    | 'IMAGE'
    | 'IMAGE_COMPONENT'
    | 'SERVICEACCOUNT';

function fillDeploymentSearchQuery(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page: number,
    perPage: number
): string {
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const queryObject: {
        pagination: Pagination;
        query?: string;
    } = {
        pagination: getPaginationParams({ page, perPage, sortOption }),
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
    sortOption: ApiSortOption,
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
    sortOption: ApiSortOption,
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

export type ListDeploymentWithProcessInfo = {
    deployment: ListDeployment;
    baselineStatuses: ContainerNameAndBaselineStatus[];
};

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

export type DeploymentWithRisk = {
    deployment: Deployment;
    risk: Risk;
};
