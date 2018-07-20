import axios from 'axios';
import queryString from 'query-string';

const baseUrl = '/v1/networkgraph';

/**
 * Fetches nodes and links for the environment graph.
 * Returns response with nodes and links
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchEnvironmentGraph(filters) {
    const params = queryString.stringify({
        ...filters
    });
    return axios.get(`${baseUrl}?${params}`).then(response => ({
        response: response.data
    }));
}

/**
 * Fetches node details for a given ID.
 *
 * @param {!string} id
 * @returns {Promise<Object, Error>}
 */
export function fetchNode(id) {
    return axios.get(`${baseUrl}/${id}`).then(response => ({
        response: response.data
    }));
}
