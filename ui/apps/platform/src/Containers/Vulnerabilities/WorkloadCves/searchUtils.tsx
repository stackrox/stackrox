import qs from 'qs';

import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import {
    VulnerabilitySeverity,
    VulnerabilityState,
    isVulnerabilityState,
    vulnerabilitySeverities,
} from 'types/cve.proto';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import { searchValueAsArray, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { ensureExhaustive } from 'utils/type.utils';

import {
    FixableStatus,
    QuerySearchFilter,
    VulnerabilitySeverityLabel,
    isFixableStatus,
    isVulnerabilitySeverityLabel,
} from './types';

export type EntityTab = 'CVE' | 'Image' | 'Deployment';

export type WorkloadCvesSearch = {
    vulnerabilityState: VulnerabilityState;
    entityTab?: EntityTab;
    s?: SearchFilter;
};

export function parseWorkloadCvesOverviewSearchString(search: string): WorkloadCvesSearch {
    const { vulnerabilityState } = qs.parse(search, { ignoreQueryPrefix: true });

    return {
        vulnerabilityState: isVulnerabilityState(vulnerabilityState)
            ? vulnerabilityState
            : 'OBSERVED',
    };
}

export function getOverviewCvesPath(workloadCvesSearch: WorkloadCvesSearch): string {
    return `${vulnerabilitiesWorkloadCvesPath}${getQueryString(workloadCvesSearch)}`;
}

export function getEntityPagePath(
    workloadCveEntity: EntityTab,
    id: string,
    queryOptions?: qs.ParsedQs
): string {
    const queryString = getQueryString(queryOptions);
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
 * Parses an open `SearchFilter` obtained from the URL into a restricted `SearchFilter` that
 * matches the fields and values expected by the backend.
 */
export function parseQuerySearchFilter(rawSearchFilter: SearchFilter): QuerySearchFilter {
    const cleanSearchFilter: QuerySearchFilter = {};

    // SearchFilter values that can be directly translated over to the backend equivalent
    const unprocessedSearchKeys = ['CVE', 'IMAGE', 'DEPLOYMENT', 'NAMESPACE', 'CLUSTER'] as const;
    unprocessedSearchKeys.forEach((key) => {
        cleanSearchFilter[key] = searchValueAsArray(rawSearchFilter[key]);
    });

    const fixable = searchValueAsArray(rawSearchFilter.Fixable);

    cleanSearchFilter.Fixable =
        fixable.length > 0
            ? fixable.filter(isFixableStatus).map(fixableStatusToFixability)
            : undefined;

    const severity = searchValueAsArray(rawSearchFilter.Severity);

    cleanSearchFilter.Severity =
        severity.length > 0
            ? severity.filter(isVulnerabilitySeverityLabel).map(severityLabelToSeverity)
            : undefined;

    return cleanSearchFilter;
}

// Given a search filter, determine which severities should be hidden from the user
export function getHiddenSeverities(
    querySearchFilter: QuerySearchFilter
): Set<VulnerabilitySeverity> {
    return querySearchFilter.Severity
        ? new Set(vulnerabilitySeverities.filter((s) => !querySearchFilter.Severity?.includes(s)))
        : new Set([]);
}

export function getHiddenStatuses(querySearchFilter: QuerySearchFilter): Set<FixableStatus> {
    const hiddenStatuses = new Set<FixableStatus>([]);
    const fixableFilters = querySearchFilter?.Fixable ?? [];

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

// Returns a search filter string that scopes results to a Vulnerability state (e.g. 'OBSERVED')
export function getVulnStateScopedQueryString(
    searchFilter: QuerySearchFilter,
    vulnerabilityState?: VulnerabilityState // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
): string {
    const vulnerabilityStateFilter = vulnerabilityState
        ? { 'Vulnerability State': vulnerabilityState }
        : {};
    return getRequestQueryStringForSearchFilter({
        ...searchFilter,
        ...vulnerabilityStateFilter,
    });
}
