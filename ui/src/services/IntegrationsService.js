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
    return axios.get(getPath(source, 'fetch')).then(response => ({
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
    return axios.put(`${getPath(source, 'save')}/${data.id}`, data);
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
 * Tests an integration by source.
 *
 * @returns {Promise<Object, Error>}
 */
export function testIntegration(source, data) {
    return axios.post(`${getPath(source, 'test')}/test`, data);
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
    return Promise.all(ids.map(id => deleteIntegration(source, id)));
}

/**
 * Triggers a DB backup
 *
 * @returns {Promise<Object, Error>}
 */
export function triggerBackup(id) {
    return axios.post(`${getPath('backups', 'trigger')}/${id}`);
}
