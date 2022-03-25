import { useState, useEffect } from 'react';
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useQuery } from '@apollo/client';

import { selectors } from 'reducers';
import { CLUSTER_WITH_NAMESPACES } from 'queries/cluster';

type SelectorState = { selectedClusterId: string | null; selectedNamespaceFilters: string[] };
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
    // If the selectedClusterId has not been set yet, do not run the gql query
    const queryOptions = selectedClusterId
        ? { variables: { id: selectedClusterId } }
        : { skip: true };

    const { loading, error, data } = useQuery<NamespaceMetadataResp, { id: string }>(
        CLUSTER_WITH_NAMESPACES,
        queryOptions
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
