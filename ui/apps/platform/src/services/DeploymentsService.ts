import queryString from 'qs';
import { normalize } from 'normalizr';

import searchOptionsToQuery, { RestSearchOption } from 'services/searchOptionsToQuery';
import { Deployment, ListDeployment } from 'types/deployment.proto';
import { ContainerNameAndBaselineStatus } from 'types/processBaseline.proto';
import { Risk } from 'types/risk.proto';
import {
    ORCHESTRATOR_COMPONENT_KEY,
    orchestratorComponentOption,
} from 'Containers/Navigation/OrchestratorComponentsToggle';
import axios from './instance';
import { deployment as deploymentSchema, deploymentDetail } from './schemas';

const deploymentsUrl = '/v1/deploymentswithprocessinfo';
const deploymentByIdUrl = '/v1/deployments';
const deploymentWithRiskUrl = '/v1/deploymentswithrisk';
const deploymentsCountUrl = '/v1/deploymentscount';

function shouldHideOrchestratorComponents() {
    // for openshift filterting toggle
    return localStorage.getItem(ORCHESTRATOR_COMPONENT_KEY) !== 'true';
}

/**
 * Fetches list of registered deployments.
 */
export function fetchDeployments(
    options: RestSearchOption[] = [],
    sortOption: Record<string, string>,
    page: number,
    pageSize: number
): Promise<ListDeploymentWithProcessInfo[]> {
    const offset = page * pageSize;
    let searchOptions: RestSearchOption[] = options;
    if (shouldHideOrchestratorComponents()) {
        searchOptions = [...options, ...orchestratorComponentOption];
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
        .get<{ deployments: ListDeploymentWithProcessInfo[] }>(`${deploymentsUrl}?${params}`)
        .then((response) => response.data.deployments);
}

export type ListDeploymentWithProcessInfo = {
    deployment: ListDeployment;
    baselineStatuses: ContainerNameAndBaselineStatus[];
};

/**
 * Fetches count of registered deployments.
 */
export function fetchDeploymentsCount(options: RestSearchOption[]): Promise<number> {
    let searchOptions: RestSearchOption[] = options;
    if (shouldHideOrchestratorComponents()) {
        searchOptions = [...options, ...orchestratorComponentOption];
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
        .then((response) => response.data.count);
}

/**
 * Fetches a deployment by its ID.
 */
export function fetchDeployment(id: string): Promise<Deployment> {
    if (!id) {
        throw new Error('Deployment ID must be specified');
    }
    return axios.get<Deployment>(`${deploymentByIdUrl}/${id}`).then((response) => response.data);
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

/**
 * Fetches list of registered deployments.
 *
 * TODO: Delete after (its only call in) deploymentSagas has been deleted.
 */
export function fetchDeploymentsLegacy(options: RestSearchOption[]): Promise<{
    response: {
        entities: { deployment: Record<string, ListDeploymentWithProcessInfo> };
        result: string[];
    };
}> {
    let searchOptions = options;
    if (shouldHideOrchestratorComponents()) {
        searchOptions = [...options, ...orchestratorComponentOption];
    }
    const query = searchOptionsToQuery(searchOptions);
    const params = queryString.stringify({ query }, { arrayFormat: 'repeat' });
    return axios
        .get<{ deployments: ListDeploymentWithProcessInfo[] }>(`${deploymentsUrl}?${params}`)
        .then((response) => ({
            response: normalize(response.data.deployments, [deploymentSchema]),
        }));
}

/**
 * Fetches a deployment by its ID.
 *
 * TODO: Delete after (its only call in) deploymentSagas has been deleted.
 */
export function fetchDeploymentLegacy(id: string): Promise<{
    response: {
        entities: { deployment: Record<string, Deployment> };
        result: string;
    };
}> {
    if (!id) {
        throw new Error('Deployment ID must be specified');
    }
    return axios.get<Deployment>(`${deploymentByIdUrl}/${id}`).then((response) => ({
        response: normalize({ deployment: response.data }, deploymentDetail),
    }));
}
