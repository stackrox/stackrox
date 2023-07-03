import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import useURLSearch from 'hooks/useURLSearch';
import { getScopeHierarchyFromSearch } from '../utils/hierarchyUtils';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

/**
 * Returns the current scope hierarchy from the URL search params.
 */
export function useScopeHierarchy(): NetworkScopeHierarchy {
    const { searchFilter } = useURLSearch();
    const { clusters } = useFetchClustersForPermissions(['NetworkGraph', 'Deployment']);

    return (
        getScopeHierarchyFromSearch(searchFilter, clusters) ?? {
            cluster: {
                id: '',
                name: '',
            },
            namespaces: [],
            deployments: [],
            remainingQuery: {},
        }
    );
}
