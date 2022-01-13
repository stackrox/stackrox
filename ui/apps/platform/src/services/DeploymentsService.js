import queryString from 'qs';
import { normalize } from 'normalizr';

import searchOptionsToQuery from 'services/searchOptionsToQuery';
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
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of deployments (as defined in .proto)
 */
export function fetchDeployments(options = [], sortOption, page, pageSize) {
    const offset = page * pageSize || 0;
    let searchOptions = options;
    if (shouldHideOrchestratorComponents()) {
        searchOptions = [...options, ...orchestratorComponentOption];
    }
    const query = searchOptionsToQuery(searchOptions);
    const queryObject = {
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
    return axios.get(`${deploymentsUrl}?${params}`).then((response) => response.data.deployments);
}

/**
 * Fetches count of registered deployments.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of deployments (as defined in .proto)
 */
export function fetchDeploymentsCount(options) {
    let searchOptions = options;
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
    return axios.get(`${deploymentsCountUrl}?${params}`).then((response) => response.data.count);
}

/**
 * Fetches a deployment by its ID.
 *
 * @returns {Promise<Object, Error>} fulfilled with a deployment object (as defined in .proto)
 */
export function fetchDeployment(id) {
    if (!id) {
        throw new Error('Deployment ID must be specified');
    }
    return axios.get(`${deploymentByIdUrl}/${id}`).then((response) => response.data);
}

/**
 * Fetches a deployment and its risk by deployment ID.
 *
 * @returns {Promise<Object, Error>} fulfilled with a composite object containing deployment and risk (as defined in .proto)
 */
export function fetchDeploymentWithRisk(id) {
    if (!id) {
        throw new Error('Deployment ID must be specified');
    }
    return axios.get(`${deploymentWithRiskUrl}/${id}`).then((response) => response.data);
}

/**
 * Fetches list of registered deployments.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of deployments (as defined in .proto)
 */
export function fetchDeploymentsLegacy(options) {
    let searchOptions = options;
    if (shouldHideOrchestratorComponents()) {
        searchOptions = [...options, ...orchestratorComponentOption];
    }
    const query = searchOptionsToQuery(searchOptions);
    const params = queryString.stringify({ query }, { arrayFormat: 'repeat' });
    return axios.get(`${deploymentsUrl}?${params}`).then((response) => ({
        response: normalize(response.data.deployments, [deploymentSchema]),
    }));
}

/**
 * Fetches a deployment by its ID.
 *
 * @returns {Promise<Object, Error>} fulfilled with a deployment object (as defined in .proto)
 */
export function fetchDeploymentLegacy(id) {
    if (!id) {
        throw new Error('Deployment ID must be specified');
    }
    return axios.get(`${deploymentByIdUrl}/${id}`).then((response) => ({
        response: normalize({ deployment: response.data }, deploymentDetail),
    }));
}
