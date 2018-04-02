import axios from 'axios';

/**
 * Fetches summary counts.
 * @returns {Promise<Object, Error>} fulfilled with response
 */

export default function fetchSummary() {
    const countsUrl = '/v1/summary/counts';
    return axios.get(countsUrl).then(response => ({
        data: response.data
    }));
}
