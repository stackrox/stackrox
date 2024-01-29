import { SearchFilter } from 'types/search';
import { SortOption } from 'types/table';

import { Pagination } from './types';

export type DiscoveredCluster = {
    id: string;
};

export function countDiscoveredClusters(/* filter */): Promise<number> {
    return Promise.resolve(0);
}

export function listDiscoveredClusters(/* arg */): Promise<DiscoveredCluster[]> {
    return Promise.resolve([]);
}

export function getListDiscoveredClustersArg({ page, perPage, searchFilter, sortOption }) {
    const filter = getDiscoveredClustersFilter(searchFilter);
    const pagination: Pagination = { limit: perPage, offset: page - 1, sortOption };
    return { filter, pagination };
}

// Given searchFilter from useURLSearch hook, return validated filter argument.

export function getDiscoveredClustersFilter(searchFilter: SearchFilter) {
    return {
        ...searchFilter, // TODO
    };
}

export function hasDiscoveredClustersFilter(searchFilter: SearchFilter) {
    return Object.keys(searchFilter).length !== 0; // TODO
}

// For useURLSort hook.

export const sortFields = [];
export const defaultSortOption: SortOption = {
    field: 'TODO',
    direction: 'asc',
};
