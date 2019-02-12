import axios from 'axios';
import queryString from 'qs';

const baseUrl = '/v1/search';

/**
 * Fetches search options
 *
 * @param {!string} query
 * @returns {Promise<Object, Error>} fulfilled with options response
 */
export function fetchOptions(query = '') {
    return axios.get(`${baseUrl}/metadata/options?${query}`).then(response => {
        const options = response.data.options.map(option => ({
            value: `${option}:`,
            label: `${option}:`,
            type: 'categoryOption'
        }));
        return { options };
    });
}

/**
 * Fetches search results
 *
 * @param {!string} query
 * @returns {Promise<Object, Error>} fulfilled with options response
 */
export function fetchGlobalSearchResults(filters) {
    const params = queryString.stringify({ ...filters }, { arrayFormat: 'repeat' });
    return axios.get(`${baseUrl}?${params}`).then(response => ({
        response: response.data
    }));
}
