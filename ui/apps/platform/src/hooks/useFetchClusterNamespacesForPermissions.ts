import { useState, useEffect } from 'react';
import {
    NamespaceForClusterAndPermissions,
    getNamespacesForClusterAndPermissions,
} from 'services/RolesService';
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
    loading: true,
    error: '',
    namespaces: [] as Namespace[],
};

function useFetchClusterNamespacesForPermissions(
    permissions: string[],
    selectedClusterId?: string
) {
    const [namespaceResponse, setNamespaceResponse] = useState<NamespaceResponse>(emptyResponse);

    useEffect(() => {
        if (selectedClusterId) {
            getNamespacesForClusterAndPermissions(selectedClusterId, permissions)
                .then((data) => {
                    const responseNamespaces = data.namespaces;
                    const namespaces: Namespace[] = [];
                    responseNamespaces.forEach(
                        (rspNamespace: NamespaceForClusterAndPermissions) => {
                            const namespace: Namespace = {} as Namespace;
                            namespace.id = rspNamespace.id;
                            namespace.name = rspNamespace.name;
                            namespaces.push(namespace);
                        }
                    );
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
    }, [permissions, selectedClusterId]);

    return namespaceResponse;
}

export default useFetchClusterNamespacesForPermissions;
