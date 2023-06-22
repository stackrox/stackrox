import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import useURLSearch from 'hooks/useURLSearch';
import { NetworkScopeHierarchy, getScopeHierarchyFromSearch } from '../utils/hierarchyUtils';

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
