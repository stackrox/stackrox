import type { SearchFilter } from 'types/search';

export const alertKeys = {
    all: ['alerts'] as const,
    lists: () => [...alertKeys.all, 'list'] as const,
    list: (filters: SearchFilter) => [...alertKeys.lists(), filters] as const,
    details: () => [...alertKeys.all, 'detail'] as const,
    detail: (id: string) => [...alertKeys.details(), id] as const,
    counts: () => [...alertKeys.all, 'count'] as const,
    count: (filters: SearchFilter) => [...alertKeys.counts(), filters] as const,
    summaryCounts: (filters: SearchFilter) => [...alertKeys.all, 'summary', filters] as const,
};

export const imageKeys = {
    all: ['images'] as const,
    lists: () => [...imageKeys.all, 'list'] as const,
    list: (filters: SearchFilter) => [...imageKeys.lists(), filters] as const,
    details: () => [...imageKeys.all, 'detail'] as const,
    detail: (id: string) => [...imageKeys.details(), id] as const,
    atMostRisk: (filters: SearchFilter) => [...imageKeys.all, 'at-most-risk', filters] as const,
    aging: (filters: SearchFilter) => [...imageKeys.all, 'aging', filters] as const,
};

export const clusterKeys = {
    all: ['clusters'] as const,
    lists: () => [...clusterKeys.all, 'list'] as const,
    list: (filters: SearchFilter) => [...clusterKeys.lists(), filters] as const,
    details: () => [...clusterKeys.all, 'detail'] as const,
    detail: (id: string) => [...clusterKeys.details(), id] as const,
};

export const deploymentKeys = {
    all: ['deployments'] as const,
    lists: () => [...deploymentKeys.all, 'list'] as const,
    list: (filters: SearchFilter) => [...deploymentKeys.lists(), filters] as const,
    details: () => [...deploymentKeys.all, 'detail'] as const,
    detail: (id: string) => [...deploymentKeys.details(), id] as const,
    atMostRisk: (filters: SearchFilter) =>
        [...deploymentKeys.all, 'at-most-risk', filters] as const,
};

export const cveKeys = {
    all: ['cves'] as const,
    lists: () => [...cveKeys.all, 'list'] as const,
    list: (filters: SearchFilter) => [...cveKeys.lists(), filters] as const,
    details: () => [...cveKeys.all, 'detail'] as const,
    detail: (id: string) => [...cveKeys.details(), id] as const,
};

export const complianceKeys = {
    all: ['compliance'] as const,
    standards: () => [...complianceKeys.all, 'standards'] as const,
    results: (filters: SearchFilter) => [...complianceKeys.all, 'results', filters] as const,
};

export const summaryKeys = {
    all: ['summary'] as const,
    counts: () => [...summaryKeys.all, 'counts'] as const,
};
