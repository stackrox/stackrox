import { useState, useEffect } from 'react';
import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { getNamespacesForClusterAndPermissions, ScopeObject } from 'services/RolesService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type SelectorState = { selectedClusterId: string | null; selectedNamespaceFilters: string[] };
type SelectorResult = SelectorState;

const selector = createStructuredSelector<SelectorState, SelectorResult>({
    selectedClusterId: selectors.getSelectedNetworkClusterId,
    selectedNamespaceFilters: selectors.getSelectedNamespaceFilters,
});

type Response = {
    loading: boolean;
    error: string;
    availableNamespaceFilters: string[];
};

const emptyResponse = {
    loading: false,
    error: '',
    availableNamespaceFilters: [],
};

function useNamespaceFilters() {
    const [response, setResponse] = useState<Response>(emptyResponse);
    const { selectedClusterId, selectedNamespaceFilters } = useSelector<
        SelectorState,
        SelectorResult
    >(selector);

    useEffect(() => {
        setResponse({
            loading: true,
            error: '',
            availableNamespaceFilters: [],
        });
        const permissions = ['NetworkGraph'];
        if (selectedClusterId) {
            getNamespacesForClusterAndPermissions(selectedClusterId, permissions)
                .then((data) => {
                    const responseNamespaces = data.namespaces;
                    const namespaces: string[] = [];
                    responseNamespaces.forEach((namespace: ScopeObject) => {
                        namespaces.push(namespace.name);
                    });
                    setResponse({
                        loading: false,
                        error: '',
                        availableNamespaceFilters: namespaces,
                    });
                })
                .catch((error) => {
                    const message = getAxiosErrorMessage(error);
                    const errorMessage =
                        message || 'An unknown error occurred while getting the list of clusters';

                    setResponse({
                        loading: false,
                        error: errorMessage,
                        availableNamespaceFilters: [],
                    });
                });
        }
    }, [selectedClusterId]);

    const { loading, error, availableNamespaceFilters } = response;
    return {
        loading,
        error,
        availableNamespaceFilters,
        selectedNamespaceFilters,
    };
}

export default useNamespaceFilters;
