import qs from 'qs';
import { cloneDeep } from 'lodash';

import { vulnerabilitiesNodeCvesPath, vulnerabilitiesPlatformCvesPath } from 'routePaths';
import {
    VulnerabilitySeverity,
    VulnerabilityState,
    vulnerabilitySeverities,
} from 'types/cve.proto';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import { searchValueAsArray, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { ensureExhaustive } from 'utils/type.utils';

import { ensureStringArray } from 'utils/ensure';

import {
    nodeSearchFilterConfig,
    nodeComponentSearchFilterConfig,
    imageSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
} from '../searchFilterConfig';

import {
    FixableStatus,
    NodeEntityTab,
    PlatformEntityTab,
    QuerySearchFilter,
    VulnerabilitySeverityLabel,
    WorkloadEntityTab,
    isFixableStatus,
    isVulnerabilitySeverityLabel,
} from '../types';

export type OverviewPageSearch = {
    s?: SearchFilter;
} & (
    | { entityTab?: WorkloadEntityTab; vulnerabilityState: VulnerabilityState }
    | { entityTab?: NodeEntityTab }
    | { entityTab?: PlatformEntityTab }
);

const baseUrlForCveMap = {
    Workload: '', // base URL provided by calling context
    Node: vulnerabilitiesNodeCvesPath,
    Platform: vulnerabilitiesPlatformCvesPath,
} as const;

export function getNamespaceViewPagePath(): string {
    return 'namespace-view';
}

export function getOverviewPagePath(
    cveBase: 'Workload' | 'Node' | 'Platform',
    pageSearch: OverviewPageSearch
): string {
    return `${baseUrlForCveMap[cveBase]}${getQueryString(pageSearch)}`;
}

export function getWorkloadEntityPagePath(
    workloadCveEntity: WorkloadEntityTab,
    id: string,
    vulnerabilityState: VulnerabilityState,
    queryOptions?: qs.ParsedQs
): string {
    const queryString = getQueryString({ ...queryOptions, vulnerabilityState });
    switch (workloadCveEntity) {
        case 'CVE':
            return `cves/${id}${queryString}`;
        case 'Image':
            return `images/${id}${queryString}`;
        case 'Deployment':
            return `deployments/${id}${queryString}`;
        default:
            return ensureExhaustive(workloadCveEntity);
    }
}

export function getPlatformEntityPagePath(
    platformCveEntity: PlatformEntityTab,
    id: string,
    queryOptions?: qs.ParsedQs
): string {
    const queryString = getQueryString(queryOptions);
    switch (platformCveEntity) {
        case 'CVE':
            // We need to encode the id here due to the `#` character literal in Platform CVE IDs
            return `${vulnerabilitiesPlatformCvesPath}/cves/${encodeURIComponent(id)}${queryString}`;
        case 'Cluster':
            return `${vulnerabilitiesPlatformCvesPath}/clusters/${id}${queryString}`;
        default:
            return ensureExhaustive(platformCveEntity);
    }
}

export function getNodeEntityPagePath(
    nodeCveEntity: NodeEntityTab,
    id: string,
    queryOptions?: qs.ParsedQs
): string {
    const queryString = getQueryString(queryOptions);
    switch (nodeCveEntity) {
        case 'CVE':
            return `${vulnerabilitiesNodeCvesPath}/cves/${id}${queryString}`;
        case 'Node':
            return `${vulnerabilitiesNodeCvesPath}/nodes/${id}${queryString}`;
        default:
            return ensureExhaustive(nodeCveEntity);
    }
}

export function fixableStatusToFixability(fixableStatus: FixableStatus): 'true' | 'false' {
    return fixableStatus === 'Fixable' ? 'true' : 'false';
}

export function severityLabelToSeverity(label: VulnerabilitySeverityLabel): VulnerabilitySeverity {
    switch (label) {
        case 'Critical':
            return 'CRITICAL_VULNERABILITY_SEVERITY';
        case 'Important':
            return 'IMPORTANT_VULNERABILITY_SEVERITY';
        case 'Moderate':
            return 'MODERATE_VULNERABILITY_SEVERITY';
        case 'Low':
            return 'LOW_VULNERABILITY_SEVERITY';
        case 'Unknown':
            return 'UNKNOWN_VULNERABILITY_SEVERITY';
        default:
            return ensureExhaustive(label);
    }
}

/**
 * Parses an open `SearchFilter` obtained from the URL into a `SearchFilter` that
 * matches the fields and values expected by the backend.
 */
export function parseQuerySearchFilter(rawSearchFilter: SearchFilter): QuerySearchFilter {
    const cleanSearchFilter: QuerySearchFilter = {};

    Object.entries(rawSearchFilter).forEach(([key, value]) => {
        const values = searchValueAsArray(value);
        if (values.length > 0) {
            cleanSearchFilter[key] = values;
        }
    });

    const fixable = searchValueAsArray(rawSearchFilter.FIXABLE);

    if (fixable.length > 0) {
        cleanSearchFilter.FIXABLE = fixable.filter(isFixableStatus).map(fixableStatusToFixability);
    }

    const clusterCveFixable = searchValueAsArray(rawSearchFilter['CLUSTER CVE FIXABLE']);

    if (clusterCveFixable.length > 0) {
        cleanSearchFilter['CLUSTER CVE FIXABLE'] = clusterCveFixable
            .filter(isFixableStatus)
            .map(fixableStatusToFixability);
    }

    const severity = searchValueAsArray(rawSearchFilter.SEVERITY);

    if (severity.length > 0) {
        cleanSearchFilter.SEVERITY = severity
            .filter(isVulnerabilitySeverityLabel)
            .map(severityLabelToSeverity);
    }

    return cleanSearchFilter;
}

export function getAppliedSeverities(searchFilter: SearchFilter): VulnerabilitySeverityLabel[] {
    return ensureStringArray(searchFilter.SEVERITY).filter(isVulnerabilitySeverityLabel);
}

// Given a search filter, determine which severities should be hidden from the user
export function getHiddenSeverities(
    querySearchFilter: QuerySearchFilter
): Set<VulnerabilitySeverity> {
    return querySearchFilter.SEVERITY
        ? new Set(vulnerabilitySeverities.filter((s) => !querySearchFilter.SEVERITY?.includes(s)))
        : new Set([]);
}

export function getHiddenStatuses(querySearchFilter: QuerySearchFilter): Set<FixableStatus> {
    const hiddenStatuses = new Set<FixableStatus>([]);
    const fixableFilters = querySearchFilter?.FIXABLE ?? [];

    if (fixableFilters.length > 0) {
        if (!fixableFilters.includes('true')) {
            hiddenStatuses.add('Fixable');
        }

        if (!fixableFilters.includes('false')) {
            hiddenStatuses.add('Not fixable');
        }
    }

    return hiddenStatuses;
}

// Returns a search filter string using regex search modifiers
export function getRegexScopedQueryString(searchFilter: QuerySearchFilter): string {
    const searchFilterWithRegex = applyRegexSearchModifiers(searchFilter);

    return getRequestQueryStringForSearchFilter(searchFilterWithRegex);
}

// Returns a search filter string that scopes results to a Vulnerability state (e.g. 'OBSERVED')
export function getVulnStateScopedQueryString(
    searchFilter: QuerySearchFilter,
    vulnerabilityState: VulnerabilityState
): string {
    const searchFilterWithRegex = applyRegexSearchModifiers(searchFilter);
    const vulnerabilityStateFilter = { 'Vulnerability State': vulnerabilityState };
    return getRequestQueryStringForSearchFilter({
        ...searchFilterWithRegex,
        ...vulnerabilityStateFilter,
    });
}

export function getZeroCveScopedQueryString(searchFilter: QuerySearchFilter): string {
    return getRequestQueryStringForSearchFilter({
        'Image CVE Count': '0',
        ...applyRegexSearchModifiers(searchFilter),
    });
}

/**
 * Returns the statuses that should be used to query for exception counts given
 * the current vulnerability state.
 * @param vulnerabilityState
 * @returns ‘PENDING’ if the vulnerability state is ‘OBSERVED’, otherwise ‘APPROVED_PENDING_UPDATE’
 */
export function getStatusesForExceptionCount(
    vulnerabilityState: VulnerabilityState | undefined
): string[] {
    return vulnerabilityState === 'OBSERVED' ? ['PENDING'] : ['APPROVED_PENDING_UPDATE'];
}

/*
 Search terms that will default to regex search.

 We only convert to regex search if the search field is of type 'text' or 'autocomplete'
*/
const regexSearchOptions = [
    nodeSearchFilterConfig,
    nodeComponentSearchFilterConfig,
    imageSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
]
    .flatMap((config) => config.attributes)
    .filter(({ inputType }) => inputType === 'text' || inputType === 'autocomplete')
    .map(({ searchTerm }) => searchTerm);

/**
 * Adds the regex search modifier to the search filter for any search options that support it.
 */
export function applyRegexSearchModifiers(searchFilter: SearchFilter): SearchFilter {
    const regexSearchFilter = cloneDeep(searchFilter);

    Object.entries(regexSearchFilter).forEach(([key, value]) => {
        if (regexSearchOptions.some((option) => option.toLowerCase() === key.toLowerCase())) {
            regexSearchFilter[key] = searchValueAsArray(value).map((val) => `r/${val}`);
        }
    });

    return regexSearchFilter;
}
