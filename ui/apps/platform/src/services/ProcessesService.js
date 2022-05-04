import axios from './instance';

const baseUrl = '/v1/processes';
const processesInBaselineUrl = '/v1/processbaselines';

/**
 * Fetches Processes for a given deployment ID.
 * Returns normalized response with policy entity extracted.
 *
 * @param {!string} deploymentId
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchProcesses(deploymentId) {
    return axios
        .get(`${baseUrl}/deployment/${deploymentId}/grouped/container`)
        .then((response) => ({
            response: response.data,
        }));
}

/**
 * Fetches container specific excluded scopes by deployment id and container id.
 *
 * @param {!string} query
 * @returns {Promise<Object, Error>} fulfilled
 */
export function fetchProcessesInBaseline(query) {
    return axios.get(`${processesInBaselineUrl}/key?${query}`).then((response) => ({
        data: response.data,
    }));
}

/**
 * Lock/Unlock container specific process excluded scope by deployment id and container id.
 *
 * @param {!array} processes
 * @returns {Promise<Object, Error>} fulfilled
 */
export function lockUnlockProcesses(processes) {
    return axios.put(`${processesInBaselineUrl}/lock`, processes);
}

/**
 * Add/Delete container specific processes excluded scope by deployment id and container id.
 *
 * @param {!array} processes
 * @returns {Promise<Object, Error>} fulfilled
 */
export function addDeleteProcesses(processes) {
    return axios.put(`${processesInBaselineUrl}`, processes);
}
