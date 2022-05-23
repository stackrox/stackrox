import { useCallback, useRef } from 'react';
import isEqual from 'lodash/isEqual';
import useURLParameter from 'hooks/useURLParameter';

function useURLNamespaceIds(defaultNamespaces: string[], allowedNamespaces: string[]) {
    const [namespacesInternal, setNamespacesInternal] = useURLParameter(
        'ns',
        defaultNamespaces.filter((ns) => allowedNamespaces.includes(ns)) || []
    );
    const namespaceRef = useRef<string[]>([]);
    const setNamespaceIds = useCallback(
        (newNamespaces: string[]) => {
            setNamespacesInternal(newNamespaces.filter((ns) => allowedNamespaces.includes(ns)));
        },
        [setNamespacesInternal, allowedNamespaces]
    );

    const filteredNamespaces: string[] = [];
    if (typeof namespacesInternal === 'string' && allowedNamespaces.includes(namespacesInternal)) {
        filteredNamespaces.push(namespacesInternal);
    } else if (namespacesInternal && Array.isArray(namespacesInternal)) {
        namespacesInternal.forEach((ns) => {
            if (typeof ns === 'string' && allowedNamespaces.includes(ns)) {
                filteredNamespaces.push(ns);
            }
        });
    }

    if (!isEqual(namespaceRef.current, filteredNamespaces)) {
        namespaceRef.current = filteredNamespaces;
    }

    return {
        namespaceIds: namespaceRef.current,
        setNamespaceIds,
    };
}

export default useURLNamespaceIds;
