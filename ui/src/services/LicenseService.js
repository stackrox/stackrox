import axios from './instance';

const url = '/v1/licenses';

/**
 * Fetches list of licenses
 *
 * @returns {Promise<Object, Error>} fulfilled with array of roles
 */
export function fetchLicenses() {
    return axios.get(`${url}/list?active=true`).then(response => ({
        response: response.data
    }));
}

/**
 * Adds a license
 *
 * @returns {Promise<Object, Error>}
 */
export function addLicense(data) {
    const payload = {
        activate: true,
        ...data
    };
    return axios.post(`${url}/add`, payload).then(response => response.data);
}
