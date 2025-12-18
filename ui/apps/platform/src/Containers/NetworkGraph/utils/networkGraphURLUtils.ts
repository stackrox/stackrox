import type { SearchFilter } from 'types/search';

export function getPropertiesForAnalytics(searchFilter: SearchFilter) {
    const cluster = searchFilter?.Cluster?.toString() ? 1 : 0;
    const namespaces = searchFilter?.Namespace?.length || 0;
    const deployments = searchFilter?.Deployment?.length || 0;

    return {
        cluster,
        namespaces,
        deployments,
    };
}
