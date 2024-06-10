import qs from 'qs';
import { cloneDeep } from 'lodash';

import {
    vulnerabilitiesNodeCvesPath,
    vulnerabilitiesPlatformCvesPath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
import {
    VulnerabilitySeverity,
    VulnerabilityState,
    vulnerabilitySeverities,
} from 'types/cve.proto';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import { searchValueAsArray, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { ensureExhaustive } from 'utils/type.utils';

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
import { regexSearchOptions } from '../searchOptions';

export type OverviewPageSearch = {
    s?: SearchFilter;
} & (
    | { entityTab?: WorkloadEntityTab; vulnerabilityState: VulnerabilityState }
    | { entityTab?: NodeEntityTab }
    | { entityTab?: PlatformEntityTab }
);

const baseUrlForCveMap = {
    Workload: vulnerabilitiesWorkloadCvesPath,
    Node: vulnerabilitiesNodeCvesPath,
    Platform: vulnerabilitiesPlatformCvesPath,
} as const;

export function getOverviewPagePath(
    cveBase: 'Workload' | 'Node' | 'Platform',
    pageSearch: OverviewPageSearch
): string {
    return `${baseUrlForCveMap[cveBase]}${getQueryString(pageSearch)}`;
}

export function getWorkloadEntityPagePath(
    workloadCveEntity: WorkloadEntityTab,
    id: string,
    vulnerabilityState: VulnerabilityState | undefined, // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    queryOptions?: qs.ParsedQs
): string {
    const queryString = getQueryString({ ...queryOptions, vulnerabilityState });
    switch (workloadCveEntity) {
        case 'CVE':
            return `${vulnerabilitiesWorkloadCvesPath}/cves/${id}${queryString}`;
        case 'Image':
            return `${vulnerabilitiesWorkloadCvesPath}/images/${id}${queryString}`;
        case 'Deployment':
            return `${vulnerabilitiesWorkloadCvesPath}/deployments/${id}${queryString}`;
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
            return `${vulnerabilitiesPlatformCvesPath}/cves/${id}${queryString}`;
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

    const severity = searchValueAsArray(rawSearchFilter.SEVERITY);

    if (severity.length > 0) {
        cleanSearchFilter.SEVERITY = severity
            .filter(isVulnerabilitySeverityLabel)
            .map(severityLabelToSeverity);
    }

    return cleanSearchFilter;
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
    vulnerabilityState?: VulnerabilityState // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
): string {
    const searchFilterWithRegex = applyRegexSearchModifiers(searchFilter);
    const vulnerabilityStateFilter = vulnerabilityState
        ? { 'Vulnerability State': vulnerabilityState }
        : {};
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
