import type { SearchFilter } from 'types/search';

/**
 * Merges the active deployment state filter into a SearchFilter for REST queries.
 * When the feature flag is disabled, the search filter is returned unchanged.
 */
export function withActiveDeploymentFilter(
    searchFilter: SearchFilter,
    isEnabled: boolean
): SearchFilter {
    if (!isEnabled) {
        return searchFilter;
    }
    return { ...searchFilter, 'Deployment State': ['DEPLOYMENT_STATE_ACTIVE'] };
}

/**
 * Appends the active deployment state clause to a GraphQL query string.
 * When the feature flag is disabled, the query string is returned unchanged.
 */
export function withActiveDeploymentQuery(query: string, isEnabled: boolean): string {
    if (!isEnabled) {
        return query;
    }
    const filter = 'Deployment State:DEPLOYMENT_STATE_ACTIVE';
    return query ? `${query}+${filter}` : filter;
}
