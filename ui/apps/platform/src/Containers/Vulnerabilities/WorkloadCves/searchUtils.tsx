import qs from 'qs';
import { cloneDeep } from 'lodash';

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
import {
    IMAGE_CVE_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
    DEPLOYMENT_SEARCH_OPTION,
    NAMESPACE_SEARCH_OPTION,
    CLUSTER_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    regexSearchOptions,
} from '../searchOptions';

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
    const unprocessedSearchKeys = [
        IMAGE_CVE_SEARCH_OPTION.value,
        IMAGE_SEARCH_OPTION.value,
        DEPLOYMENT_SEARCH_OPTION.value,
        NAMESPACE_SEARCH_OPTION.value,
        CLUSTER_SEARCH_OPTION.value,
        COMPONENT_SEARCH_OPTION.value,
        COMPONENT_SOURCE_SEARCH_OPTION.value,
    ] as const;
    unprocessedSearchKeys.forEach((key) => {
        const values = searchValueAsArray(rawSearchFilter[key]);
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
        if (regexSearchOptions.some((option) => option === key)) {
            regexSearchFilter[key] = searchValueAsArray(value).map((val) => `r/${val}`);
        }
    });

    return regexSearchFilter;
}
