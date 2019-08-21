import queryString from 'qs';
import { normalize } from 'normalizr';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import axios from './instance';

import { deployment as deploymentSchema, deploymentDetail } from './schemas';

const deploymentsUrl = '/v1/deploymentswithprocessinfo';
const deploymentByIdUrl = '/v1/deployments';
const deploymentsCountUrl = '/v1/deploymentscount';

/**
 * Fetches list of registered deployments.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of deployments (as defined in .proto)
 */
export function fetchDeployments(options, sortOption, page, pageSize) {
    const offset = page * pageSize;
    const query = searchOptionsToQuery(options);
    const params = queryString.stringify(
        {
            query,
            pagination: {
                offset,
                limit: pageSize,
                sortOption
            }
        },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios.get(`${deploymentsUrl}?${params}`).then(response => response.data.deployments);
}

/**
 * Fetches count of registered deployments.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of deployments (as defined in .proto)
 */
export function fetchDeploymentsCount(options) {
    const params = queryString.stringify(
        { query: searchOptionsToQuery(options) },
        { arrayFormat: 'repeat' }
    );
    return axios.get(`${deploymentsCountUrl}?${params}`).then(response => response.data.count);
}

/**
 * Fetches a deployment by its ID.
 *
 * @returns {Promise<Object, Error>} fulfilled with a deployment object (as defined in .proto)
 */
export function fetchDeployment(id) {
    if (!id) throw new Error('Deployment ID must be specified');
    return axios.get(`${deploymentByIdUrl}/${id}`).then(response => response.data);
}

/**
 * Fetches list of registered deployments.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of deployments (as defined in .proto)
 */
export function fetchDeploymentsLegacy(options) {
    const params = queryString.stringify(
        { query: searchOptionsToQuery(options) },
        { arrayFormat: 'repeat' }
    );
    return axios
        .get(`${deploymentsUrl}?${params}`)
        .then(response => ({ response: normalize(response.data.deployments, [deploymentSchema]) }));
}

/**
 * Fetches a deployment by its ID.
 *
 * @returns {Promise<Object, Error>} fulfilled with a deployment object (as defined in .proto)
 */
export function fetchDeploymentLegacy(id) {
    if (!id) throw new Error('Deployment ID must be specified');
    return axios.get(`${deploymentByIdUrl}/${id}`).then(response => ({
        response: normalize({ deployment: response.data }, deploymentDetail)
    }));
}
