import axios from 'axios';

/**
 * Fetches list of registered clusters.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of clusters (as defined in .proto)
 */
export function fetchClusters() {
    const clustersUrl = '/v1/clusters';
    return axios.get(clustersUrl).then(response => ({
        response: response.data
    }));
}

/**
 * Fetches list of registered clusters.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of clusters (as defined in .proto)
 */
export function deleteCluster(id) {
    const clustersUrl = '/v1/clusters/';
    return axios.delete(`${clustersUrl}${id}`);
}

/**
 * Sends a POST to create a new cluster
 *
 * @returns {Promise<Object[], Error>} fulfilled with id of cluster (as defined in .proto)
 */
export function createCluster(data) {
    const clustersUrl = '/v1/clusters';
    return axios.post(`${clustersUrl}`, data);
}
