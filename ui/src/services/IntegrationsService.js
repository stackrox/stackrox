import axios from 'axios';

const sourceMap = {
    dnrIntegrations: '/v1/dnrintegrations',
    imageIntegrations: '/v1/imageintegrations',
    notifiers: '/v1/notifiers'
};

/**
 * Fetches list of registered integrations based on source.
 *
 * @returns {Promise<Object, Error>} fulfilled with array of the integration source
 */
export function fetchIntegration(source) {
    return axios.get(sourceMap[source]).then(response => ({
        response: response.data
    }));
}

/**
 * Saves an integration by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function saveIntegration(source, data) {
    if (!data.id) throw new Error('Integration entity must have an id to be saved');
    return axios.put(`${sourceMap[source]}/${data.id}`, data);
}

/**
 * Creates an integration by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function createIntegration(source, data) {
    return axios.post(sourceMap[source], data);
}

/**
 * Tests an integration by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function testIntegration(source, data) {
    return axios.post(`${sourceMap[source]}/test`, data);
}

/**
 * Deletes an integration by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function deleteIntegration(source, id) {
    return axios.delete(`${sourceMap[source]}/${id}`);
}

/**
 * Deletes a list of integrations by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function deleteIntegrations(source, ids = []) {
    return Promise.all(ids.map(id => deleteIntegration(source, id)));
}
