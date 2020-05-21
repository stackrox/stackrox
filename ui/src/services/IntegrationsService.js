import axios from './instance';

function getPath(type, action) {
    switch (type) {
        case 'imageIntegrations':
            return '/v1/imageintegrations';
        case 'notifiers':
            return '/v1/notifiers';
        case 'backups':
            return '/v1/externalbackups';
        case 'authPlugins':
            if (action === 'test') {
                return '/v1/scopedaccessctrl';
            }
            if (action === 'fetch') {
                return '/v1/scopedaccessctrl/configs';
            }
            return '/v1/scopedaccessctrl/config';
        default:
            return '';
    }
}

/**
 * Fetches list of registered integrations based on source.
 *
 * @returns {Promise<Object, Error>} fulfilled with array of the integration source
 */
export function fetchIntegration(source) {
    const path = getPath(source, 'fetch');
    return axios.get(path).then((response) => ({
        response: response.data,
    }));
}

/**
 * Saves an integration by source. If it can potentially use stored credentials, use the
 * updatePassword option to determine if you should
 *
 * @param {string} source - The source of the integration
 * @param {Object} data - The form data
 * @param {Object} options - Contains a field like "updatePassword" to determine what API to use
 * @returns {Promise<Object, Error>}
 */
export function saveIntegration(source, data, options = {}) {
    if (!data.id) throw new Error('Integration entity must have an id to be saved');
    const { updatePassword } = options;
    // if the integration is not one that could possibly have stored credentials, use the previous API
    if (updatePassword === null) return axios.put(`${getPath(source, 'save')}/${data.id}`, data);
    // if it does, format the request data and use the new API
    const integration = {
        config: data,
        updatePassword,
    };
    return axios.patch(`${getPath(source, 'save')}/${data.id}`, integration);
}

/**
 * Creates an integration by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function createIntegration(source, data) {
    return axios.post(getPath(source, 'create'), data);
}

/**
 * Tests an integration by source. If it can potentially use stored credentials, use the
 * updatePassword option to determine if you should
 *
 * @param {string} source - The source of the integration
 * @param {Object} data - The form data
 * @param {Object} options - Contains a field like "updatePassword" to determine what API to use
 * @returns {Promise<Object, Error>}
 */
export function testIntegration(source, data, options = {}) {
    const { updatePassword } = options;
    // if the integration is not one that could possibly have stored credentials, use the previous API
    if (updatePassword === null) return axios.post(`${getPath(source, 'test')}/test`, data);
    // if it does, format the request data and use the new API
    const integration = {
        config: data,
        updatePassword,
    };
    return axios.post(`${getPath(source, 'test')}/test/updated`, integration);
}

/**
 * Deletes an integration by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function deleteIntegration(source, id) {
    return axios.delete(`${getPath(source, 'delete')}/${id}`);
}

/**
 * Deletes a list of integrations by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function deleteIntegrations(source, ids = []) {
    return Promise.all(ids.map((id) => deleteIntegration(source, id)));
}

/**
 * Triggers a DB backup
 *
 * @returns {Promise<Object, Error>}
 */
export function triggerBackup(id) {
    return axios.post(`${getPath('backups', 'trigger')}/${id}`);
}
