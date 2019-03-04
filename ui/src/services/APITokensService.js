import axios from './instance';

const url = '/v1/apitokens';

/**
 * Fetches list of (unrevoked) API tokens.
 *
 * @returns {Promise<Object, Error>} fulfilled with array of the integration source
 */
export function fetchAPITokens() {
    return axios.get(`${url}?revoked=false`).then(response => ({
        response: response.data
    }));
}

export function generateAPIToken(data) {
    return axios.post(`${url}/generate`, data).then(response => ({
        response: response.data
    }));
}

export function revokeAPIToken(id) {
    return axios.patch(`${url}/revoke/${id}`);
}

export function revokeAPITokens(ids) {
    return Promise.all(ids.map(revokeAPIToken));
}
