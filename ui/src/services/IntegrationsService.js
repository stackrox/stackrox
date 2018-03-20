import axios from 'axios';

const sourceMap = {
    notifiers: '/v1/notifiers',
    registries: '/v1/registries',
    scanners: '/v1/scanners'
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
    return data.id !== undefined && data.id !== ''
        ? axios.put(`${sourceMap[source]}/${data.id}`, data)
        : axios.post(sourceMap[source], data);
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
 * Deletes a list of integrations by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function deleteIntegration(source, data) {
    return axios.delete(`${sourceMap[source]}/${data.id}`);
}
