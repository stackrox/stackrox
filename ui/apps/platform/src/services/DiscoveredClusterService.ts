import qs from 'qs';

import type { SearchFilter } from 'types/search';
import type { SortOption } from 'types/table';

import { getPaginationParams } from 'utils/searchUtils';
import axios from './instance';

import type { Pagination } from './types';

export type DiscoveredCluster = {
    // UUIDv5 generated deterministically from the tuple (metadata.id, metadata.type, source.id).
    id: string;
    metadata: DiscoveredClusterMetadata;
    status: DiscoveredClusterStatus;
    source: DiscoveredClusterCloudSource;
};

export type DiscoveredClusterMetadata = {
    // id under which the cluster is registered with the cloud provider.
    // Matches storage.ClusterMetadata.id for secured clusters.
    id: string;
    // name under which the cluster is registered with the cloud provider.
    // Matches storage.ClusterMetadata.name for secured clusters.
    name: string;
    // Matches storage.ClusterMetadata.type for secured clusters.
    type: DiscoveredClusterType;
    providerType: DiscoveredClusterProviderType;
    region: string;
    // When the cluster was first discovered by the cloud source.
    firstDiscoveredAt: string; // ISO 8601
};

// type

// Order of items is for search filter options.
export const types = ['AKS', 'ARO', 'EKS', 'GKE', 'OCP', 'OSD', 'ROSA', 'UNSPECIFIED'] as const; // for isType function

export type DiscoveredClusterType = (typeof types)[number];

// providerType

export type DiscoveredClusterProviderType =
    | 'PROVIDER_TYPE_AWS'
    | 'PROVIDER_TYPE_AZURE'
    | 'PROVIDER_TYPE_GCP'
    | 'PROVIDER_TYPE_UNSPECIFIED';

// status

// Order of items is for search filter options.
export const statuses = ['STATUS_SECURED', 'STATUS_UNSECURED', 'STATUS_UNSPECIFIED'] as const; // for isStatus function

export type DiscoveredClusterStatus = (typeof statuses)[number];

// source

export type DiscoveredClusterCloudSource = {
    id: string;
};

// endpoints

const basePath = '/v1/discovered-clusters';
const countPath = '/v1/count/discovered-clusters';

export type CountDiscoveredClustersResponse = {
    count: number; // int32
};

export function countDiscoveredClusters(filter: DiscoveredClustersFilter): Promise<number> {
    const params = qs.stringify({ filter }, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<CountDiscoveredClustersResponse>(`${countPath}?${params}`)
        .then((response) => response.data.count);
}

export type GetDiscoveredClusterResponse = {
    cluster: DiscoveredCluster;
};

export function getDiscoveredCluster(id: string): Promise<DiscoveredCluster> {
    return axios
        .get<GetDiscoveredClusterResponse>(`${basePath}/${id}`)
        .then((response) => response.data.cluster);
}

export type ListDiscoveredClustersRequest = {
    filter: DiscoveredClustersFilter;
    pagination: Pagination;
};

export type ListDiscoveredClustersResponse = {
    clusters: DiscoveredCluster[];
};

export function listDiscoveredClusters(
    arg: ListDiscoveredClustersRequest
): Promise<DiscoveredCluster[]> {
    const params = qs.stringify(arg, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<ListDiscoveredClustersResponse>(`${basePath}?${params}`)
        .then((response) => response.data.clusters);
}

export function getListDiscoveredClustersArg({
    page,
    perPage,
    searchFilter,
    sortOption,
}): ListDiscoveredClustersRequest {
    const filter = getDiscoveredClustersFilter(searchFilter);
    const pagination = getPaginationParams({ page, perPage, sortOption });
    return { filter, pagination };
}

export type DiscoveredClustersFilter = {
    names?: string[];
    sourceIds?: string[];
    statuses?: DiscoveredClusterStatus[];
    types?: DiscoveredClusterType[];
};

// Given searchFilter from useURLSearch hook, return validated filter argument.

// See proto/storage/discovered_cluster.proto
export const nameField = 'Cluster'; // metadata.name is for filter and sort
export const sourceIdField = 'Integration ID'; // source.id
export const statusField = 'Cluster Status'; // status
export const typeField = 'Cluster Type'; // metadata.type

export function getDiscoveredClustersFilter(searchFilter: SearchFilter): DiscoveredClustersFilter {
    return {
        names: getValues(searchFilter[nameField]),
        sourceIds: getValues(searchFilter[sourceIdField]),
        statuses: getStatuses(searchFilter[statusField]),
        types: getTypes(searchFilter[typeField]),
    };
}

export function hasDiscoveredClustersFilter(searchFilter: SearchFilter) {
    const { names, sourceIds, statuses, types } = getDiscoveredClustersFilter(searchFilter);
    return hasItems(names) || hasItems(sourceIds) || hasItems(statuses) || hasItems(types);
}

function hasItems(arg: unknown[] | undefined) {
    return Array.isArray(arg) && arg.length !== 0;
}

type SearchFilterValue = string | string[] | undefined;

// names and sourceIds

function getValues(arg: SearchFilterValue): string[] | undefined {
    if (typeof arg === 'string') {
        return [arg];
    }

    if (Array.isArray(arg)) {
        return arg;
    }

    return undefined;
}

export function replaceSearchFilterNames(searchFilter: SearchFilter, names: string[] | undefined) {
    return { ...searchFilter, [nameField]: names };
}

// statuses

function getStatuses(arg: SearchFilterValue): DiscoveredClusterStatus[] | undefined {
    if (typeof arg === 'string' && isStatus(arg)) {
        return [arg];
    }

    if (Array.isArray(arg)) {
        return arg.filter(isStatus);
    }

    return undefined;
}

export function isStatus(arg: string): arg is DiscoveredClusterStatus {
    return statuses.some((level) => level === arg);
}

export function replaceSearchFilterStatuses(
    searchFilter: SearchFilter,
    statuses: DiscoveredClusterStatus[] | undefined
): SearchFilter {
    return { ...searchFilter, [statusField]: statuses };
}

// types

function getTypes(arg: SearchFilterValue): DiscoveredClusterType[] | undefined {
    if (typeof arg === 'string' && isType(arg)) {
        return [arg];
    }

    if (Array.isArray(arg)) {
        return arg.filter(isType);
    }

    return undefined;
}

export function isType(arg: string): arg is DiscoveredClusterType {
    return types.some((level) => level === arg);
}

export function replaceSearchFilterTypes(
    searchFilter: SearchFilter,
    types: DiscoveredClusterType[] | undefined
): SearchFilter {
    return { ...searchFilter, [typeField]: types };
}

// For useURLSort hook.

export const firstDiscoveredAtField = 'Cluster Discovered Time';

export const sortFields = [nameField];
export const defaultSortOption: SortOption = {
    field: nameField,
    direction: 'asc',
};
