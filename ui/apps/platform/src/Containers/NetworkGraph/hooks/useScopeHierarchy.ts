import useURLSearch from 'hooks/useURLSearch';
import { ClusterScopeObject } from 'services/RolesService';
import { getScopeHierarchyFromSearch } from '../utils/hierarchyUtils';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

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

    return getScopeHierarchyFromSearch(searchFilter, availableClusters) ?? emptyScopeHierarchy;
}
