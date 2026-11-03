import qs from 'qs';

import searchOptionsToQuery from 'services/searchOptionsToQuery';
import type { RestSearchOption } from 'services/searchOptionsToQuery';
import type { ClusterResponse, ClustersResponse } from 'types/clusterService.proto';
import axios from './instance';
import type { Empty } from './types';

const clustersUrl = '/v1/clusters';
const clusterInitUrl = '/v1/cluster-init';
const upgradesUrl = '/v1/sensorupgrades';

export type ClusterLabels = Record<string, string>;

export type Cluster = {
    id: string;
    name: string;
};

/**
 * Fetches list of registered clusters.
 */
export function fetchClusters(options?: RestSearchOption[]): Promise<Cluster[]> {
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

/**
 * Start a cluster cert rotation.
 */
export function rotateClusterCerts(id: string): Promise<Empty> {
    return axios.post(`${upgradesUrl}/rotateclustercerts/${id}`);
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

export type UserAttribute = {
    key: string;
    value: string;
};

export type ClusterRegistrationSecret = {
    id: string;
    name: string;
    createdAt: string;
    createdBy: {
        id: string;
        authProviderId: string;
        attributes: UserAttribute[];
    };
    expiresAt: string;
    // Protobuf uint64 is serialized as a JSON string. Parse with parseInt when a numeric value is needed.
    maxRegistrations: string;
    registrationsInitiated: string[];
    registrationsCompleted: string[];
};

export function fetchClusterRegistrationSecrets(): Promise<{ items: ClusterRegistrationSecret[] }> {
    return axios
        .get<{ items: ClusterRegistrationSecret[] }>(`${clusterInitUrl}/crs`)
        .then((response) => {
            return response.data ?? { items: [] };
        });
}

export type GenerateClusterRegistrationSecretResponse = {
    meta?: ClusterRegistrationSecret;
    crs?: string; // bytes
};

export type GenerateClusterRegistrationSecretExtendedRequest = {
    name: string;
    validUntil?: string;
    validFor?: string;
    maxRegistrations?: string;
};

export function generateClusterRegistrationSecretExtended(
    data: GenerateClusterRegistrationSecretExtendedRequest
): Promise<GenerateClusterRegistrationSecretResponse> {
    return axios
        .post<{
            meta: ClusterRegistrationSecret;
            crs: string;
        }>(`${clusterInitUrl}/crs-extended`, data)
        .then((response) => {
            return response.data;
        });
}

export type ClusterRegistrationSecretRevocationError = {
    id: string;
    error: string;
};

export type ClusterRegistrationSecretRevokeResponse = {
    crsRevocationErrors: ClusterRegistrationSecretRevocationError[];
    revokedIds: string[];
};

export function revokeClusterRegistrationSecrets(
    ids: string[]
): Promise<ClusterRegistrationSecretRevokeResponse> {
    return axios
        .patch<ClusterRegistrationSecretRevokeResponse>(`${clusterInitUrl}/crs/revoke`, {
            ids,
        })
        .then((response) => {
            return response.data;
        });
}
