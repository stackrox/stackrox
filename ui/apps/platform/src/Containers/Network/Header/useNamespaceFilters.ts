import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { useFetchClusterNamespacesForPermissions } from 'hooks/useFetchClusterNamespacesForPermissions';

type SelectorState = { selectedClusterId: string | null; selectedNamespaceFilters: string[] };
type SelectorResult = SelectorState;

const selector = createStructuredSelector<SelectorState, SelectorResult>({
    selectedClusterId: selectors.getSelectedNetworkClusterId,
    selectedNamespaceFilters: selectors.getSelectedNamespaceFilters,
});

function useNamespaceFilters() {
    const { selectedClusterId, selectedNamespaceFilters } = useSelector<
        SelectorState,
        SelectorResult
    >(selector);

    const { loading, error, namespaces } = useFetchClusterNamespacesForPermissions(
        ['NetworkGraph', 'Deployment'],
        selectedClusterId
    );
    const availableNamespaceFilters = namespaces.map((namespace) => namespace.name);

    return {
        loading,
        error,
        availableNamespaceFilters,
        selectedNamespaceFilters,
    };
}

export default useNamespaceFilters;
