import axios from 'axios';
import { normalize } from 'normalizr';

import { saveFile } from 'services/DownloadService';
import { cluster as clusterSchema } from './schemas';

const clustersUrl = '/v1/clusters';

/**
 * Fetches list of registered clusters.
 *
 * @returns {Promise<Object, Error>} fulfilled with normalized list of clusters
 */
export function fetchClusters() {
    return axios.get(clustersUrl).then(response => ({
        response: normalize(response.data, { clusters: [clusterSchema] })
    }));
}

/**
 * Fetches cluster by its ID.
 *
 * @returns {Promise<Object, Error>} fulfilled with normalized cluster data
 */
export function fetchCluster(id) {
    return axios.get(`${clustersUrl}/${id}`).then(response => ({
        response: normalize(response.data, { cluster: clusterSchema })
    }));
}

/**
 * Deletes cluster given the cluster ID.
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export function deleteCluster(id) {
    return axios.delete(`${clustersUrl}/${id}`);
}

/**
 * Deletes clusters given a list of cluster IDs.
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export function deleteClusters(ids = []) {
    return Promise.all(ids.map(id => deleteCluster(id)));
}

/**
 * Creates or updates a cluster given the cluster fields.
 *
 * @returns {Promise<Object, Error>} fulfilled with a saved cluster data
 */
export function saveCluster(cluster) {
    const promise = cluster.id
        ? axios.put(`${clustersUrl}/${cluster.id}`, cluster)
        : axios.post(clustersUrl, cluster);
    return promise.then(response => ({
        response: normalize(response.data, { cluster: clusterSchema })
    }));
}

/**
 * Downloads cluster YAML configuration.
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export function downloadClusterYaml(clusterId) {
    return saveFile({
        method: 'post',
        url: '/api/extensions/clusters/zip',
        data: { id: clusterId }
    });
}
