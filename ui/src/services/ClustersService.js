import axios from 'axios';

/**
 * Fetches list of registered clusters.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of clusters (as defined in .proto)
 */
export default function fetchClusters() {
    const clustersUrl = '/v1/clusters';
    return axios.get(clustersUrl).then(response => response.data.clusters);
}
