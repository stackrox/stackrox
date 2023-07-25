import cloneDeep from 'lodash/cloneDeep';

import useURLSearch from 'hooks/useURLSearch';
import { ClusterScopeObject } from 'services/RolesService';
import { SearchFilter } from 'types/search';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

export function getScopeHierarchyFromSearch(
    searchFilter: SearchFilter,
    clusters: ClusterScopeObject[]
): NetworkScopeHierarchy | null {
    const urlCluster = searchFilter.Cluster;
    if (!urlCluster || Array.isArray(urlCluster)) {
        return null;
    }

    const cluster = clusters.find((cl) => cl.name === urlCluster);
    if (!cluster) {
        return null;
    }

    const workingQuery = { ...searchFilter };
    delete workingQuery.Cluster;

    const hierarchy: NetworkScopeHierarchy = {
        cluster,
        namespaces: [],
        deployments: [],
        remainingQuery: workingQuery,
    };

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
        delete hierarchy.remainingQuery.Deployment;
    }

    return hierarchy;
}

const emptyScopeHierarchy = {
    cluster: {
        id: '',
        name: '',
    },
    namespaces: [],
    deployments: [],
    remainingQuery: {},
};

/**
 * Returns the current scope hierarchy from the URL search params.
 */
export function useScopeHierarchy(availableClusters: ClusterScopeObject[]): NetworkScopeHierarchy {
    const { searchFilter } = useURLSearch();

    return (
        getScopeHierarchyFromSearch(searchFilter, availableClusters) ??
        cloneDeep(emptyScopeHierarchy)
    );
}
