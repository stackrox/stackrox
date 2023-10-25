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

import { FixableStatus, QuerySearchFilter } from './types';

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

/**
 * Parses an open `SearchFilter` obtained from the URL into a restricted `SearchFilter` that
 * matches the fields and values expected by the backend.
 */
export function parseQuerySearchFilter(rawSearchFilter: SearchFilter): QuerySearchFilter {
    const cleanSearchFilter: QuerySearchFilter = {};

    // SearchFilter values that can be directly translated over to the backend equivalent
    const unprocessedSearchKeys = ['CVE', 'IMAGE', 'DEPLOYMENT', 'NAMESPACE', 'CLUSTER'] as const;
    unprocessedSearchKeys.forEach((key) => {
        if (rawSearchFilter[key]) {
            cleanSearchFilter[key] = searchValueAsArray(rawSearchFilter[key]);
        }
    });

    if (rawSearchFilter.Fixable) {
        const rawFixable = searchValueAsArray(rawSearchFilter.Fixable);
        const cleanFixable: ('true' | 'false')[] = [];

        rawFixable.forEach((status) => {
            if (status === 'Fixable') {
                cleanFixable.push('true');
            } else if (status === 'Not fixable') {
                cleanFixable.push('false');
            }
        });

        // TODO We are explicitly excluding "Fixable" from the search filter until this functionality is re-enabled
        // cleanSearchFilter.Fixable = cleanFixable;
    }

    if (rawSearchFilter.Severity) {
        const rawSeverities = searchValueAsArray(rawSearchFilter.Severity);
        cleanSearchFilter.Severity = [];

        rawSeverities.forEach((rs) => {
            if (rs === 'Critical') {
                cleanSearchFilter.Severity?.push('CRITICAL_VULNERABILITY_SEVERITY');
            } else if (rs === 'Important') {
                cleanSearchFilter.Severity?.push('IMPORTANT_VULNERABILITY_SEVERITY');
            } else if (rs === 'Moderate') {
                cleanSearchFilter.Severity?.push('MODERATE_VULNERABILITY_SEVERITY');
            } else if (rs === 'Low') {
                cleanSearchFilter.Severity?.push('LOW_VULNERABILITY_SEVERITY');
            }
        });
    }

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
