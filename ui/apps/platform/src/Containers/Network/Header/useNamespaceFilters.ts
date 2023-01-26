import { useState, useEffect } from 'react';
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import useFetchNamespacesForClusterAndPermission from 'hooks/useFetchClusterNamespacesForPermission';
import { AccessLevel } from 'services/RolesService';

type SelectorState = { selectedClusterId: string | null; selectedNamespaceFilters: string[] };
type SelectorResult = SelectorState;

const selector = createStructuredSelector<SelectorState, SelectorResult>({
    selectedClusterId: selectors.getSelectedNetworkClusterId,
    selectedNamespaceFilters: selectors.getSelectedNamespaceFilters,
});

function useNamespaceFilters() {
    const [availableNamespaceFilters, setAvailableNamespaceFilters] = useState<string[]>([]);
    const { selectedClusterId, selectedNamespaceFilters } = useSelector<
        SelectorState,
        SelectorResult
    >(selector);

    const networkGraphResource = 'NetworkGraph';
    const readAccess: AccessLevel = 'READ_ACCESS';
    const result = useFetchNamespacesForClusterAndPermission(
        networkGraphResource,
        readAccess,
        selectedClusterId || '-'
    );
    const loading = result.isLoading;
    const error = result.error;

    useEffect(() => {
        // if (!data || !data.results) {
        //     return;
        // }

        // const namespaces = data.results.namespaces.map(({ metadata }) => metadata.name);
        const namespaces = result.namespaces.map(({metadata}) => metadata.name);

        setAvailableNamespaceFilters(namespaces);
    }, [result]);

    return {
        loading,
        error,
        availableNamespaceFilters,
        selectedNamespaceFilters,
    };
}

export default useNamespaceFilters;
