import { useState, useEffect } from 'react';

import {
    AccessLevel,
    NamespaceForPermission,
    getNamespacesForClusterAndPermission,
} from 'services/RolesService';
import { getAxiosErrorMessage } from '../utils/responseErrorUtils';

export type Namespace = {
    metadata: {
        id: string;
        name: string;
    };
    deploymentCount;
};

type Result = {
    isLoading: boolean;
    error: string;
    namespaces: Namespace[];
};

function useFetchNamespacesForClusterAndPermission(
    resource: string,
    access: AccessLevel,
    clusterId?: string
): Result {
    const defaultResultState = {
        namespaces: [],
        error: '',
        isLoading: true,
    };

    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        getNamespacesForClusterAndPermission({ resource, access }, clusterId)
            .then((data) => {
                const responseNamespaces = data.namespaces;
                const namespaces: Namespace[] = [];
                responseNamespaces.forEach((rspNs: NamespaceForPermission) => {
                    const namespace: Namespace = {} as Namespace;
                    namespace.metadata.id = rspNs.id;
                    namespace.metadata.name = rspNs.name;
                    namespaces.push(namespace)
                });
                setResult({
                    namespaces: namespaces || [],
                    error: '',
                    isLoading: false,
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({
                    namespaces: [],
                    error: errorMessage,
                    isLoading: false,
                });
            });
    }, []);

    return result;
}

export default useFetchNamespacesForClusterAndPermission;
