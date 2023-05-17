import sortBy from 'lodash/sortBy';
import { ensureExhaustive } from 'utils/type.utils';
import { EntityTab } from './types';

export const defaultImageSortFields = [
    'Image',
    'Operating system',
    'Deployment count',
    'Age',
    'Scan time',
];

export const imagesDefaultSort = {
    field: 'Image',
    direction: 'desc',
} as const;

export const defaultCVESortFields = ['Deployment', 'Cluster', 'Namespace'];

export const CVEsDefaultSort = {
    field: 'CVE',
    direction: 'asc',
} as const;

export const defaultDeploymentSortFields = ['Deployment', 'Cluster', 'Namespace'];

export const deploymentsDefaultSort = {
    field: 'Deployment',
    direction: 'asc',
} as const;

export function getDefaultSortOption(entityTab: EntityTab) {
    switch (entityTab) {
        case 'CVE':
            return CVEsDefaultSort;
        case 'Deployment':
            return deploymentsDefaultSort;
        case 'Image':
            return imagesDefaultSort;
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
