import sortBy from 'lodash/sortBy';
import { ensureExhaustive } from 'utils/type.utils';
import { SortAggregate, SortOption } from 'types/table';
import { WorkloadEntityTab } from '../types';

export const aggregateByCVSS: SortAggregate = {
    aggregateFunc: 'max',
};

export const aggregateByDistinctCount: SortAggregate = {
    aggregateFunc: 'count',
    distinct: 'true',
};

export const aggregateByCreatedTime: SortAggregate = {
    aggregateFunc: 'min',
};

/**
 * Get the available sort fields for a given Workload CVE entity
 * @param entityTab The chosen entity
 * @returns The available sort fields
 */
export function getWorkloadSortFields(entityTab: WorkloadEntityTab): string[] {
    switch (entityTab) {
        case 'CVE':
            return ['CVE', 'CVSS', 'Image Sha', 'CVE Created Time'];
        case 'Image':
            return ['Image', 'Image OS', 'Image created time', 'Image scan time'];
        case 'Deployment':
            return ['Deployment', 'Cluster', 'Namespace', 'Created'];
        default:
            return ensureExhaustive(entityTab);
    }
}

/**
 *  Get the default table sort option for a given Workload CVE entity
 *
 * @param entityTab The chosen entity
 * @returns The default sort option
 */
export function getDefaultWorkloadSortOption(entityTab: WorkloadEntityTab): SortOption {
    switch (entityTab) {
        case 'CVE':
            return { field: 'CVSS', aggregateBy: aggregateByCVSS, direction: 'desc' };
        case 'Deployment':
            return { field: 'Deployment', direction: 'asc' };
        case 'Image':
            return { field: 'Image created time', direction: 'asc' };
        default:
            return ensureExhaustive(entityTab);
    }
}

/**
 * Gets the default sort option for a given Workload CVE entity when the user is viewing
 * the images and deployments that are observed to have zero CVEs.
 *
 * @param entityTab The chosen entity
 * @returns The default sort option
 */
export function getDefaultZeroCveSortOption(entityTab: WorkloadEntityTab): SortOption {
    switch (entityTab) {
        case 'CVE':
            return { field: 'CVE', direction: 'asc' };
        case 'Deployment':
            return { field: 'Deployment', direction: 'asc' };
        case 'Image':
            return { field: 'Image', direction: 'asc' };
        default:
            return ensureExhaustive(entityTab);
    }
}

/**
 * The priority order of supported operating systems when displaying summary and link information.
 */
const distroPriorityMap = {
    rhel: 1,
    centos: 2,
    ubuntu: 3,
    debian: 4,
    alpine: 5,
    amzn: 6,
    other: Infinity,
} as const;
const distroKeys = Object.keys(distroPriorityMap) as (keyof typeof distroPriorityMap)[];
export type Distro = (typeof distroKeys)[number];

// Given an array of objects with an operatingSystem field, return the sorted by operating system priority order.
// The priority is defined by matching the prefix of the operating system string with the prefixes in the priority list. Items
// that do not match anything in the list should have the worst priority.
export function sortCveDistroList<Summary extends { operatingSystem: string }>(
    distros: Summary[]
): (Summary & { distro: Distro })[] {
    const withDistroKeys = distros.map((distro) => ({
        ...distro,
        distro: distroKeys.find((p) => distro.operatingSystem.startsWith(p)) ?? 'other',
    }));
    return sortBy(withDistroKeys, ({ distro }) => distroPriorityMap[distro]);
}

export function getScoreVersionsForTopCVSS(
    topCvss: number,
    scores: { cvss: number; scoreVersion: string }[]
): string[] {
    const scoreVersions = scores
        .filter(({ cvss }) => cvss.toFixed(1) === topCvss.toFixed(1))
        .map(({ scoreVersion }) => scoreVersion);
    return Array.from(new Set(scoreVersions)).sort();
}
