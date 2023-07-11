import qs from 'qs';

import searchOptionsToQuery, { RestSearchOption } from 'services/searchOptionsToQuery';
import { saveFile } from 'services/DownloadService';
import {
    ClusterDefaultsResponse,
    ClusterResponse,
    ClustersResponse,
} from 'types/clusterService.proto';
import axios from './instance';
import { Empty } from './types';

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
        return response.data;
    });
}

/*
 * Fetch secured cluster and its retention information.
 */
export function fetchClusterWithRetentionInformation(id: string): Promise<ClusterResponse> {
    return axios.get<ClusterResponse>(`${clustersUrl}/${id}`).then((response) => {
        return response.data;
    });
}

export type AutoUpgradeConfig = {
    enableAutoUpgrade: boolean;
    autoUpgradeFeature: 'SUPPORTED' | 'NOT_SUPPORTED';
};

/**
 * Checks is auto upgrade is supported
 */
export function isAutoUpgradeSupported(autoUpgradeConfig: AutoUpgradeConfig) {
    return autoUpgradeConfig.autoUpgradeFeature === 'SUPPORTED';
}

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
export function saveAutoUpgradeConfig(config: AutoUpgradeConfig): Promise<Empty> {
    const wrappedObject = { config: { enableAutoUpgrade: config.enableAutoUpgrade } };
    return axios.post<Empty>(autoUpgradeConfigUrl, wrappedObject).then((response) => response.data);
}

/**
 * Manually start a sensor upgrade given the cluster ID.
 */
export function upgradeCluster(id: string): Promise<Empty> {
    return axios.post(`${manualUpgradeUrl}/${id}`);
}

/**
 * Start a cluster cert rotation.
 */
export function rotateClusterCerts(id: string): Promise<Empty> {
    return axios.post(`${upgradesUrl}/rotateclustercerts/${id}`);
}

/**
 * Manually start a sensor upgrade for an array of clusters.
 */
export function upgradeClusters(ids = []): Promise<Empty[]> {
    return Promise.all(ids.map((id) => upgradeCluster(id)));
}

/**
 * Deletes cluster given the cluster ID. Returns an empty object.
 */
export function deleteCluster(id: string): Promise<Empty> {
    return axios.delete(`${clustersUrl}/${id}`);
}

/**
 * Deletes clusters given a list of cluster IDs.
 */
export function deleteClusters(ids: string[] = []): Promise<Empty[]> {
    return Promise.all(ids.map((id) => deleteCluster(id)));
}

/**
 * Creates or updates a cluster given the cluster fields.
 */
export function saveCluster(cluster: Cluster) {
    if (cluster.id) {
        return axios
            .put<ClusterResponse>(`${clustersUrl}/${cluster.id}`, cluster)
            .then((response) => response.data);
    }
    return axios.post<ClusterResponse>(clustersUrl, cluster).then((response) => response.data);
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
