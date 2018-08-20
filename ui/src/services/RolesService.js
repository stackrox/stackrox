import axios from 'axios';

const url = '/v1/roles';

/**
 * Fetches list of roles
 *
 * @returns {Promise<Object, Error>} fulfilled with array of roles
 */
export default function fetchRoles() {
    return axios.get(url).then(response => ({
        response: response.data
    }));
}
