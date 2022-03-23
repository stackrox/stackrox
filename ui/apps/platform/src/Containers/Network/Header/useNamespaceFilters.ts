import { useState, useEffect } from 'react';
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useQuery } from '@apollo/client';

import { selectors } from 'reducers';
import { CLUSTER_WITH_NAMESPACES } from 'queries/cluster';

type SelectorState = { selectedClusterId: string; selectedNamespaceFilters: string[] };
type SelectorResult = SelectorState;

const selector = createStructuredSelector<SelectorState, SelectorResult>({
    selectedClusterId: selectors.getSelectedNetworkClusterId,
    selectedNamespaceFilters: selectors.getSelectedNamespaceFilters,
});

// TODO This is a minimum expected return type - do we have clearly defined typings elsewhere?
type NamespaceMetadataResp = {
    id: string;
    results: {
        namespaces: {
            metadata: {
                name: string;
            };
        }[];
    };
};

function useNamespaceFilters() {
    const [availableNamespaceFilters, setAvailableNamespaceFilters] = useState<string[]>([]);
    const { selectedClusterId, selectedNamespaceFilters } = useSelector<
        SelectorState,
        SelectorResult
    >(selector);
    const queryVariables = {
        variables: {
            id: selectedClusterId,
        },
    };
    const { loading, error, data } = useQuery<NamespaceMetadataResp, { id: string }>(
        CLUSTER_WITH_NAMESPACES,
        queryVariables
    );

    useEffect(() => {
        if (!data || !data.results) {
            return;
        }

        const namespaces = data.results.namespaces.map(({ metadata }) => metadata.name);

        setAvailableNamespaceFilters(namespaces);
    }, [data]);

    return {
        loading,
        error,
        availableNamespaceFilters,
        selectedNamespaceFilters,
    };
}

export default useNamespaceFilters;
