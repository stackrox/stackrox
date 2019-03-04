import axios from './instance';

/**
 * Fetches summary counts.
 * @returns {Promise<Object, Error>} fulfilled with response
 */

export default function fetchSummaryCounts() {
    const countsUrl = '/v1/summary/counts';
    return axios.get(countsUrl).then(response => ({
        response: response.data
    }));
}
