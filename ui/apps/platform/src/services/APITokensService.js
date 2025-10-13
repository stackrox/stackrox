import axios from './instance';

const url = '/v1/apitokens';

/**
 * Fetches list of (unrevoked) API tokens.
 *
 * @returns {Promise<Object, Error>} fulfilled with array of the integration source
 */
export function fetchAPITokens() {
    return axios.get(`${url}?revoked=false`).then((response) => ({
        response: response.data,
    }));
}

export function fetchAllowedRoles() {
    return axios.get(`${url}/generate/allowed-roles`).then((response) => response.data.roleNames);
}

export function generateAPIToken(data) {
    const options = {
        method: 'post',
        url: `${url}/generate`,
        data,
        // extend timeout to one minute, for https://stack-rox.atlassian.net/browse/ROX-5183
        timeout: 60000,
    };

    return axios(options).then((response) => ({
        response: response.data,
    }));
}

export function revokeAPIToken(id) {
    return axios.patch(`${url}/revoke/${id}`);
}

export function revokeAPITokens(ids) {
    return Promise.all(ids.map(revokeAPIToken));
}
