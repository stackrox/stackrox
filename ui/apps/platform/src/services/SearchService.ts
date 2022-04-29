import queryString from 'qs';
import { SearchCategory } from 'constants/searchOptions';
import { SearchEntry } from 'types/search';
import axios from './instance';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';

const baseUrl = '/v1/search';
const autoCompleteURL = `${baseUrl}/autocomplete`;

type OptionResponse = { options: string[] };
type AutocompleteResponse = { values: string[] };

/**
 * Fetches search options
 *
 * @param {!string} query
 * @returns {Promise<Object, Error>} fulfilled with options response
 */
export function fetchOptions(query = '') {
    return axios.get<OptionResponse>(`${baseUrl}/metadata/options?${query}`).then((response) => {
        const options =
            response?.data?.options?.map(
                (option): SearchEntry => ({
                    value: `${option}:`,
                    label: `${option}:`,
                    type: 'categoryOption',
                })
            ) ?? {};
        return { options };
    });
}

/*
 * Get search options for category.
 */
export function getSearchOptionsForCategory(
    searchCategory: SearchCategory
): CancellableRequest<string[]> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<OptionResponse>(`${baseUrl}/metadata/options?categories=${searchCategory}`, {
                signal,
            })
            .then((response) => response?.data?.options ?? [])
    );
}

/**
 * Fetches search results
 *
 * @param {!string} query
 * @returns {Promise<Object, Error>} fulfilled with options response
 */
export function fetchGlobalSearchResults(filters) {
    const params = queryString.stringify({ ...filters }, { arrayFormat: 'repeat' });
    // Note for future TS narrowing: the return type for this data consists of many types of entities
    return axios.get(`${baseUrl}?${params}`).then((response) => ({
        response: response.data,
    }));
}

// Fetches the autocomplete response.
export function fetchAutoCompleteResults({ query, categories }) {
    const params = queryString.stringify({ query, categories }, { arrayFormat: 'repeat' });
    return axios
        .get<AutocompleteResponse>(`${autoCompleteURL}?${params}`)
        .then((response) => response?.data?.values || []);
}
