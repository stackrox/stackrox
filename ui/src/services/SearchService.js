import axios from 'axios';

const baseUrl = '/v1/search';

/**
 * Fetches search options
 *
 * @param {!string} query
 * @returns {Promise<Object, Error>} fulfilled with options response
 */
export default function fetchOptions(query = '') {
    return axios
        .get(`${baseUrl}/metadata/options?${query}`)
        .then(response => {
            const options = response.data.options.map(option => ({
                value: `${option}:`,
                label: `${option}:`,
                type: 'categoryOption'
            }));
            options.unshift({
                value: `Has:`,
                label: `Has:`,
                type: 'categoryOption'
            });
            return { options };
        })
        .catch(() => ({ options: [] }));
}
