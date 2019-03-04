import axios from './instance';

const url = '/v1/roles';

/**
 * Fetches list of roles
 *
 * @returns {Promise<Object, Error>} fulfilled with array of roles
 */
export function fetchRoles() {
    return axios.get(url).then(response => ({
        response: response.data
    }));
}

/**
 * Creates a role.
 *
 * @returns {Promise<Object, Error>}
 */
export function createRole(data) {
    const { name } = data;
    return axios.post(`${url}/${name}`, data);
}

/**
 * Updates a role.
 *
 * @returns {Promise<Object, Error>}
 */
export function updateRole(data) {
    const { name } = data;
    return axios.put(`${url}/${name}`, data);
}

/**
 * Deletes a role.
 *
 * @returns {Promise<Object, Error>}
 */
export function deleteRole(id) {
    return axios.delete(`${url}/${id}`);
}
