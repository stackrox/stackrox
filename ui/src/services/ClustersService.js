import { normalize } from 'normalizr';

import { saveFile } from 'services/DownloadService';
import axios from './instance';
import { cluster as clusterSchema } from './schemas';

const clustersUrl = '/v1/clusters';
const upgradesUrl = '/v1/sensorupgrades';
const autoUpgradeConfigUrl = `${upgradesUrl}/config`;
const manualUpgradeUrl = `${upgradesUrl}/cluster`;

// @TODO, We may not need this API function after we migrate to a standalone Clusters page
//        Check to see if fetchClusters and fletchClustersByArray can be collapsed
//        into one function
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
 * Fetches list of registered clusters as an Array.
 *
 * @returns {Promise<Object, Error>} fulfilled with normalized list of clusters
 */
export function fetchClustersAsArray() {
    return axios.get(clustersUrl).then(response => {
        return (response.data && response.data.clusters) || [];
    });
}

/**
 * Fetches unwrapped cluster object by ID.
 *
 * @returns {Promise<Object, Error>} fulfilled with single cluster object
 */
export function getClusterById(id) {
    return axios.get(`${clustersUrl}/${id}`).then(response => {
        return (response && response.data && response.data.cluster) || null;
    });
}

/**
 * Gets the cluster autoupgrade config.
 *
 * @returns {Promise<Object, Error>} fulfilled with autoupgrade config object
 */
export function getAutoUpgradeConfig() {
    return axios.get(autoUpgradeConfigUrl).then(response => {
        return (response.data && response.data.config) || {};
    });
}

/**
 * Saves the cluster autoupgrade config.
 *
 * @returns {Promise<Object, Error>} whose only value is resolved or rejected
 */
export function saveAutoUpgradeConfig(config) {
    const wrappedObject = { config };
    return axios.post(autoUpgradeConfigUrl, wrappedObject);
}

/**
 * Manually start a sensor upgrade given the cluster ID.
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export function upgradeCluster(id) {
    return axios.post(`${manualUpgradeUrl}/${id}`);
}

/**
 * Manually start a sensor upgrade for an array of clusters.
 *
 * @returns {Promise<Object, Error>} whose only value is resolved or rejected
 */
export function upgradeClusters(ids = []) {
    return Promise.all(ids.map(id => upgradeCluster(id)));
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
