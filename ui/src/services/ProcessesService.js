import axios from './instance';

const baseUrl = '/v1/processes';

/**
 * Fetches policy details for a given policy ID.
 * Returns normalized response with policy entity extracted.
 *
 * @param {!string} policyId
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export default function fetchProcesses(deploymentId) {
    return axios.get(`${baseUrl}/deployment/${deploymentId}/grouped`).then(response => ({
        response: response.data
    }));
}
