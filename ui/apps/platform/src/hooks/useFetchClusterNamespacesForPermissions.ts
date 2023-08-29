import { useState, useEffect } from 'react';
import { NamespaceScopeObject, getNamespacesForClusterAndPermissions } from 'services/RolesService';
import { ResourceName } from 'types/roleResources';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type NamespaceResponse = {
    loading: boolean;
    error: string;
    namespaces: NamespaceScopeObject[];
};

const emptyResponse: NamespaceResponse = {
    loading: false,
    error: '',
    namespaces: [],
};

export function useFetchClusterNamespacesForPermissions(
    permissions: ResourceName[],
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
                    setNamespaceResponse({
                        loading: false,
                        error: '',
                        namespaces: data.namespaces,
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
