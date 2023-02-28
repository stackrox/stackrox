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
    selectedClusterId?: string | null
) {
    const [requestedPermissions] = useState<string[]>(permissions);
    const [namespaceResponse, setNamespaceResponse] = useState<NamespaceResponse>(emptyResponse);

    useEffect(() => {
        if (selectedClusterId) {
            setNamespaceResponse({
                loading: true,
                error: '',
                namespaces: [],
            });
            getNamespacesForClusterAndPermissions(selectedClusterId, requestedPermissions)
                .then((data) => {
                    const namespaces: Namespace[] = data.namespaces.map((ns: ScopeObject) => {
                        return {
                            id: ns.id,
                            name: ns.name,
                        } as Namespace;
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
