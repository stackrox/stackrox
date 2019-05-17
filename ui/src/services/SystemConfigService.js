import axios from './instance';

const baseUrl = '/v1/config';

/**
 * Fetches system configurations.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchSystemConfig() {
    return axios.get(baseUrl).then(({ data }) => ({
        response: data
    }));
}

/**
 * Fetches login notice and header/footer info.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchPublicConfig() {
    return axios.get(`${baseUrl}/public`).then(({ data }) => ({
        response: data
    }));
}

/**
 * Saves modified system config.
 *
 * @param {!object} config
 * @returns {Promise<AxiosResponse, Error>}
 */
export function saveSystemConfig(config) {
    return axios.put(baseUrl, config);
}
