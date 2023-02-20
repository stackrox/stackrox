import { useState, useEffect } from 'react';
import { ScopeObject, getNamespacesForClusterAndPermissions } from 'services/RolesService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type Namespace = {
    id: string;
    name: string;
};

type NamespaceResponse = {
    loading: boolean;
    error: string;
    namespaces: Namespace[];
};

const emptyResponse: NamespaceResponse = {
    loading: false,
    error: '',
    namespaces: [] as Namespace[],
};

export function useFetchClusterNamespacesForPermissions(
    permissions: string[],
    selectedClusterId?: string
) {
    const [requestedPermissions] = useState<string[]>(permissions);
    const [namespaceResponse, setNamespaceResponse] = useState<NamespaceResponse>(emptyResponse);

    useEffect(() => {
        setNamespaceResponse({
            loading: true,
            error: '',
            namespaces: [],
        });
        if (selectedClusterId) {
            getNamespacesForClusterAndPermissions(selectedClusterId, requestedPermissions)
                .then((data) => {
                    const responseNamespaces = data.namespaces;
                    const namespaces: Namespace[] = [];
                    responseNamespaces.forEach((responseNamespace: ScopeObject) => {
                        const namespace: Namespace = {} as Namespace;
                        namespace.id = responseNamespace.id;
                        namespace.name = responseNamespace.name;
                        namespaces.push(namespace);
                    });
                    setNamespaceResponse({
                        loading: false,
                        error: '',
                        namespaces,
                    });
                })
                .catch((error) => {
                    const message = getAxiosErrorMessage(error);
                    const errorMessage =
                        message || 'An unknown error occurred while getting the list of clusters';

                    setNamespaceResponse({
                        loading: false,
                        error: errorMessage,
                        namespaces: [],
                    });
                });
        }
    }, [requestedPermissions, selectedClusterId]);

    return namespaceResponse;
}
