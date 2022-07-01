import { normalize } from 'normalizr';
import qs from 'qs';

import searchOptionsToQuery, { RestSearchOption } from 'services/searchOptionsToQuery';
import { saveFile } from 'services/DownloadService';
import {
    ClusterDefaultsResponse,
    ClusterResponse,
    ClustersResponse,
} from 'types/clusterService.proto';
import axios from './instance';
import { cluster as clusterSchema } from './schemas';

const clustersUrl = '/v1/clusters';
const clusterDefaultsUrl = '/v1/cluster-defaults';
const clusterInitUrl = '/v1/cluster-init';
const upgradesUrl = '/v1/sensorupgrades';
const autoUpgradeConfigUrl = `${upgradesUrl}/config`;
const manualUpgradeUrl = `${upgradesUrl}/cluster`;

export type ClusterLabels = Record<string, string>;

export type Cluster = {
    id: string;
    name: string;
};

// @TODO, We may not need this API function after we migrate to a standalone Clusters page
//        Check to see if fetchClusters and fletchClustersByArray can be collapsed
//        into one function
/**
 * Fetches list of registered clusters.
 */
// TODO specify return type after we rewrite without normalize
// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export function fetchClusters() {
    return axios.get<{ clusters: Cluster[] }>(clustersUrl).then((response) => ({
        response: normalize(response.data, { clusters: [clusterSchema] }),
    }));
}

/**
 * Fetches list of registered clusters.
 */
export function fetchClustersAsArray(options?: RestSearchOption[]): Promise<Cluster[]> {
    let queryString = '';
    if (options && options.length !== 0) {
        const query = searchOptionsToQuery(options);
        queryString = qs.stringify(
            {
                query,
            },
            {
                addQueryPrefix: true,
                arrayFormat: 'repeat',
                allowDots: true,
            }
        );
    }
    return axios.get<{ clusters: Cluster[] }>(`${clustersUrl}${queryString}`).then((response) => {
        return response?.data?.clusters ?? [];
    });
}

/*
 * Fetch secured clusters and retention information.
 */
export function fetchClustersWithRetentionInfo(
    options?: RestSearchOption[]
): Promise<ClustersResponse> {
    let queryString = '';
    if (options && options.length !== 0) {
        const query = searchOptionsToQuery(options);
        queryString = qs.stringify(
            {
                query,
            },
            {
                addQueryPrefix: true,
                arrayFormat: 'repeat',
                allowDots: true,
            }
        );
    }
    return axios.get<ClustersResponse>(`${clustersUrl}${queryString}`).then((response) => {
        return response?.data;
    });
}

/*
 * Fetch secured cluster and its retention information by ID.
 */
export function fetchClusterWithRetentionInformationById(id: string): Promise<ClusterResponse> {
    return axios.get<ClusterResponse>(`${clustersUrl}/${id}`).then((response) => {
        return response?.data;
    });
}

export type AutoUpgradeConfig = {
    enableAutoUpgrade?: boolean;
};

/**
 * Gets the cluster autoupgrade config.
 */
export function getAutoUpgradeConfig(): Promise<AutoUpgradeConfig> {
    return axios.get<{ config: AutoUpgradeConfig }>(autoUpgradeConfigUrl).then((response) => {
        return response?.data?.config ?? {};
    });
}

/**
 * Saves the cluster autoupgrade config.
 */
export function saveAutoUpgradeConfig(config: AutoUpgradeConfig): Promise<AutoUpgradeConfig> {
    const wrappedObject = { config };
    return axios.post(autoUpgradeConfigUrl, wrappedObject);
}

/**
 * Manually start a sensor upgrade given the cluster ID.
 */
export function upgradeCluster(id: string): Promise<Record<string, never>> {
    return axios.post(`${manualUpgradeUrl}/${id}`);
}

/**
 * Start a cluster cert rotation.
 */
export function rotateClusterCerts(id: string): Promise<Record<string, never>> {
    return axios.post(`${upgradesUrl}/rotateclustercerts/${id}`);
}

/**
 * Manually start a sensor upgrade for an array of clusters.
 */
export function upgradeClusters(ids = []): Promise<Record<string, never>[]> {
    return Promise.all(ids.map((id) => upgradeCluster(id)));
}

/**
 * Fetches cluster by its ID.
 */
// TODO specify return type after we rewrite without normalize
// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export function fetchCluster(id: string) {
    return axios.get(`${clustersUrl}/${id}`).then((response) => ({
        response: normalize(response.data, { cluster: clusterSchema }),
    }));
}

/**
 * Deletes cluster given the cluster ID. Returns an empty object.
 */
export function deleteCluster(id: string): Promise<Record<string, never>> {
    return axios.delete(`${clustersUrl}/${id}`);
}

/**
 * Deletes clusters given a list of cluster IDs.
 */
export function deleteClusters(ids: string[] = []): Promise<Record<string, never>[]> {
    return Promise.all(ids.map((id) => deleteCluster(id)));
}

/**
 * Creates or updates a cluster given the cluster fields.
 */
// TODO specify return type after we rewrite without normalize
// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export function saveCluster(cluster: Cluster) {
    const promise = cluster.id
        ? axios.put(`${clustersUrl}/${cluster.id}`, cluster)
        : axios.post(clustersUrl, cluster);
    return promise.then((response) => ({
        response: normalize(response.data, { cluster: clusterSchema }),
    }));
}

/**
 * Downloads cluster YAML configuration.
 */
export function downloadClusterYaml(id: string, createUpgraderSA = false): Promise<void> {
    return saveFile({
        method: 'post',
        url: '/api/extensions/clusters/zip',
        data: { id, createUpgraderSA },
    });
}

/**
 * Downloads cluster Helm YAML configuration.
 */
export function downloadClusterHelmValuesYaml(id: string): Promise<void> {
    return saveFile({
        method: 'post',
        url: '/api/extensions/clusters/helm-config.yaml',
        data: { id },
    });
}

/*
 * Get default images for new clusters that depend on the Central configuration.
 * Also get kernelSupportAvailable for slimCollector property.
 */
export function getClusterDefaults(): Promise<ClusterDefaultsResponse> {
    return axios.get<ClusterDefaultsResponse>(clusterDefaultsUrl).then((response) => {
        return response.data;
    });
}

export type InitBundleAttribute = {
    key: string;
    value: string;
};

export type ImpactedCluster = {
    name: string;
    id: string;
};

export type ClusterInitBundle = {
    id: string;
    name: string;
    createdAt: string;
    createdBy: {
        id: string;
        authProviderId: string;
        attributes: InitBundleAttribute[];
    };
    expiresAt: string;
    impactedClusters: ImpactedCluster[];
};

export function fetchCAConfig(): Promise<{ helmValuesBundle?: string }> {
    return axios
        .get<{ helmValuesBundle: string }>(`${clusterInitUrl}/ca-config`)
        .then((response) => {
            return response?.data;
        });
}

export function fetchClusterInitBundles(): Promise<{ response: { items: ClusterInitBundle[] } }> {
    return axios
        .get<{ items: ClusterInitBundle[] }>(`${clusterInitUrl}/init-bundles`)
        .then((response) => {
            return {
                response: response.data || { items: [] },
            };
        });
}

export function generateClusterInitBundle(data: { name: string }): Promise<{
    response: { meta?: ClusterInitBundle; helmValuesBundle?: string; kubectlBundle?: string };
}> {
    return axios
        .post<{ meta: ClusterInitBundle; helmValuesBundle: string; kubectlBundle: string }>(
            `${clusterInitUrl}/init-bundles`,
            data
        )
        .then((response) => {
            return {
                response: response.data || {},
            };
        });
}

export type InitBundleRevocationError = {
    id: string;
    error: string;
    impactedClusters: ImpactedCluster[];
};

export type InitBundleRevokeResponse = {
    initBundleRevocationErrors: InitBundleRevocationError[];
    initBundleRevokedIds: string[];
};

export function revokeClusterInitBundles(
    ids: string[],
    confirmImpactedClustersIds: string[]
): Promise<InitBundleRevokeResponse> {
    return axios
        .patch<InitBundleRevokeResponse>(`${clusterInitUrl}/init-bundles/revoke`, {
            ids,
            confirmImpactedClustersIds,
        })
        .then((response) => {
            return response.data;
        });
}
