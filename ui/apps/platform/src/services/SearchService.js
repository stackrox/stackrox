import queryString from 'qs';
import axios from './instance';

const baseUrl = '/v1/search';
const autoCompleteURL = `${baseUrl}/autocomplete`;

/**
 * Fetches search options
 *
 * @param {!string} query
 * @returns {Promise<Object, Error>} fulfilled with options response
 */
export function fetchOptions(query = '') {
    return axios.get(`${baseUrl}/metadata/options?${query}`).then((response) => {
        const options =
            response?.data?.options?.map((option) => ({
                value: `${option}:`,
                label: `${option}:`,
                type: 'categoryOption',
            })) ?? {};
        return { options };
    });
}

/*
 * Get search options for category.
 */
export function getSearchOptionsForCategory(searchCategory) {
    return axios
        .get(`${baseUrl}/metadata/options?categories=${searchCategory}`)
        .then((response) => response?.data?.options ?? []);
}

/**
 * Fetches search results
 *
 * @param {!string} query
 * @returns {Promise<Object, Error>} fulfilled with options response
 */
export function fetchGlobalSearchResults(filters) {
    const params = queryString.stringify({ ...filters }, { arrayFormat: 'repeat' });
    return axios.get(`${baseUrl}?${params}`).then((response) => ({
        response: response.data,
    }));
}

// Fetches the autocomplete response.
export function fetchAutoCompleteResults({ query, categories }) {
    const params = queryString.stringify({ query, categories }, { arrayFormat: 'repeat' });
    return axios
        .get(`${autoCompleteURL}?${params}`)
        .then((response) => response?.data?.values || []);
}
