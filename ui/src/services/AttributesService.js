import axios from './instance';

const url = '/v1/usersattributes';

/**
 * Fetches list of users attributes
 *
 * @returns {Promise<Object, Error>} fulfilled with array of users attributes
 */
export default function fetchUsersAttributes() {
    return axios.get(url).then(response => ({
        response: response.data
    }));
}
