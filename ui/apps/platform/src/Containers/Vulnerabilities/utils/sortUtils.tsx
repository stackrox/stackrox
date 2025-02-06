import intersection from 'lodash/intersection';
import sortBy from 'lodash/sortBy';
import { ensureExhaustive, isNonEmptyArray, NonEmptyArray } from 'utils/type.utils';
import { SortAggregate, SortOption } from 'types/table';
import { FieldOption } from 'hooks/useURLSort';
import { ApiSortOption, SearchFilter } from 'types/search';
import {
    vulnerabilitySeverityLabels,
    VulnerabilitySeverityLabel,
    WorkloadEntityTab,
} from '../types';
import { getAppliedSeverities } from './searchUtils';

// ROX-27906 Image CVEs view cannot use search fields as sort options without providing aggregates
// aggregateByCVSS and aggregateByEPSS might become unnecessary in the future, at least for WorkloadCVEOverviewTable

export const aggregateByCVSS: SortAggregate = {
    aggregateFunc: 'max',
};

export const aggregateByEPSS: SortAggregate = {
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
export function getWorkloadCveOverviewSortFields(
    entityTab: WorkloadEntityTab
): (string | string[])[] {
    switch (entityTab) {
        case 'CVE':
            return [
                'CVE',
                [
                    'Critical Severity Count',
                    'Important Severity Count',
                    'Moderate Severity Count',
                    'Low Severity Count',
                ],
                'CVSS',
                'Image Sha',
                'CVE Created Time',
            ];
        case 'Image':
            return [
                'Image',
                [
                    'Critical Severity Count',
                    'Important Severity Count',
                    'Moderate Severity Count',
                    'Low Severity Count',
                ],
                'Image OS',
                'Image Created Time',
                'Image Scan Time',
            ];
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
export function getWorkloadCveOverviewDefaultSortOption(
    entityTab: WorkloadEntityTab,
    searchFilter?: SearchFilter
): SortOption | NonEmptyArray<SortOption> {
    // Array.prototype.map does not currently retain the arity of an input tuple, so
    // we need to cast the return value to a NonEmptyArray<SortOption>. This may be fixed
    // soon in a future version of TypeScript https://github.com/microsoft/TypeScript/issues/29841
    const appliedSeveritySortOptions = getSeveritySortOptions(
        getAppliedSeverities(searchFilter ?? {})
    ).map((o) => ({ ...o, direction: 'desc' })) as NonEmptyArray<SortOption>;

    switch (entityTab) {
        case 'CVE':
            return appliedSeveritySortOptions;
        case 'Deployment':
            return { field: 'Deployment', direction: 'asc' };
        case 'Image':
            return appliedSeveritySortOptions;
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

export function getScoreVersionsForTopNvdCVSS(
    topNvdCvss: number,
    scores: { nvdCvss: number; nvdScoreVersion: string }[]
): string[] {
    const scoreVersions = scores
        .filter(({ nvdCvss = 0 }) => nvdCvss.toFixed(1) === topNvdCvss.toFixed(1))
        .map(({ nvdScoreVersion = 'UNKNOWN_VERSION' }) => nvdScoreVersion);
    return Array.from(new Set(scoreVersions)).sort();
}

export const severitySortMap = {
    Critical: 'Critical Severity Count',
    Important: 'Important Severity Count',
    Moderate: 'Moderate Severity Count',
    Low: 'Low Severity Count',
} as const;

/**
 * Given the selected severity filters, return the sort options for the severity column. Severities
 * that are hidden will not be included in the sort options.
 *
 * @param selectedSeverities Severities that have been enabled by the user
 * @returns A non-empty array of sort options for the severity columns
 */
export function getSeveritySortOptions(
    selectedSeverities: VulnerabilitySeverityLabel[] | undefined
): NonEmptyArray<FieldOption> {
    const options = vulnerabilitySeverityLabels
        .filter((severity) => selectedSeverities?.includes(severity))
        .map((severity) => ({ field: severitySortMap[severity] }));

    if (isNonEmptyArray(options)) {
        return options;
    }

    return [
        { field: 'Critical Severity Count' },
        { field: 'Important Severity Count' },
        { field: 'Moderate Severity Count' },
        { field: 'Low Severity Count' },
    ];
}

/**
 * If the current sort option is sorting by severity, update the sort option
 * to match the applied severity filters when the filters change. This is necessary because
 * the severity multi sort is dynamic and changes based on the applied severity filters, but
 * does not update the sort fields automatically when the filters change.
 *
 * @param searchFilter The updated search filter
 * @param currentSortOption The current sort option
 * @param applySort A callback function to apply the new sort option, usually from the useURLSort hook
 */
export function syncSeveritySortOption(
    searchFilter: SearchFilter,
    currentSortOption: ApiSortOption,
    applySort: (sort: SortOption | SortOption[]) => void
) {
    // Only sync the sort option if the current sort option is sorting by severity. This
    // is determined by detecting that the current sort option is an array and that it
    // contains a field that matches a severity.
    if (
        !Array.isArray(currentSortOption) ||
        intersection(
            currentSortOption.map((s) => s.field),
            Object.values(severitySortMap)
        ).length === 0
    ) {
        return;
    }

    const appliedSeverities = getAppliedSeverities(searchFilter);
    const { reversed } = currentSortOption[0];
    const direction = reversed ? 'desc' : 'asc';
    const sortOptions = getSeveritySortOptions(appliedSeverities).map(
        (option) => ({ ...option, direction }) as const
    );
    applySort(sortOptions);
}
