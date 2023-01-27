import { SearchFilter } from 'types/search';
import { NamespaceWithDeployments } from 'hooks/useFetchNamespaceDeployments';

export function getScopeHierarchyFromSearch(searchFilter: SearchFilter) {
    const workingQuery = { ...searchFilter };
    const hierarchy: {
        cluster: string | undefined;
        namespaces: string[];
        deployments: string[];
        remainingQuery;
    } = {
        cluster: undefined,
        namespaces: [],
        deployments: [],
        remainingQuery: workingQuery,
    };

    if (searchFilter.Cluster && !Array.isArray(searchFilter.Cluster)) {
        hierarchy.cluster = searchFilter.Cluster;
        delete hierarchy.remainingQuery.Cluster;
    }

    if (searchFilter.Namespace) {
        hierarchy.namespaces = Array.isArray(searchFilter.Namespace)
            ? searchFilter.Namespace
            : [searchFilter.Namespace];
        delete hierarchy.remainingQuery.Namespace;
    }

    if (searchFilter.Deployment) {
        hierarchy.deployments = Array.isArray(searchFilter.Deployment)
            ? searchFilter.Deployment
            : [searchFilter.Deployment];
    }

    return hierarchy;
}

export function getDeploymentLookupMap(
    deploymentsByNamespace: NamespaceWithDeployments[]
): Record<string, string[]> {
    return deploymentsByNamespace.reduce<Record<string, string[]>>((acc, ns) => {
        const deployments = ns.deployments.map((deployment) => deployment.name);
        return { ...acc, [ns.metadata.name]: deployments };
    }, {});
}

export default {
    getScopeHierarchyFromSearch,
    getDeploymentLookupMap,
};
