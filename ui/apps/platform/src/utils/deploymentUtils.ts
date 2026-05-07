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
 * Replaces any existing Deployment State clause in a GraphQL query string with
 * the active deployment state filter. This avoids conflicting conditions when the
 * query already contains a deployment state filter (e.g. from the inactive images
 * base filter).
 * When the feature flag is disabled, the query string is returned unchanged.
 */
export function withActiveDeploymentQuery(query: string, isEnabled: boolean): string {
    if (!isEnabled) {
        return query;
    }
    const filter = 'Deployment State:DEPLOYMENT_STATE_ACTIVE';
    // Replace existing Deployment State terms to avoid conflicting conditions.
    const terms = query.split('+').filter((t) => !t.startsWith('Deployment State:'));
    return [...terms, filter].filter(Boolean).join('+');
}
