import { createContext, useContext, useEffect, useMemo, useRef } from 'react';
import type { ReactNode } from 'react';

import type { Empty } from 'services/types';

import { ALL_NAMESPACES_KEY } from './constants';

export type AuthScope = Empty | { namespace: string; workload?: string };

export type ScopeGetter = () => AuthScope;

type ScopeContextValue = {
    getScope: ScopeGetter;
    setScope: (scope: AuthScope) => void;
};

const ScopeContext = createContext<ScopeContextValue | null>(null);

export function ScopeProvider({ children }: { children: ReactNode }) {
    const scopeRef = useRef<AuthScope>({});

    // Store scope in a ref (not state) to avoid triggering re-renders when scope changes.
    // The scope is only read by the axios adapter at request time, so reactive updates are not needed.
    const value = useMemo<ScopeContextValue>(
        () => ({
            getScope: () => scopeRef.current,
            setScope: (scope: AuthScope) => {
                scopeRef.current = scope;
            },
        }),
        []
    );

    return <ScopeContext.Provider value={value}>{children}</ScopeContext.Provider>;
}

export function useScopeContext() {
    const context = useContext(ScopeContext);
    if (!context) {
        throw new Error('useScopeContext must be used within ScopeProvider');
    }
    return context;
}

/**
 * Sets namespace scope for all API requests. Side-effect only - does not trigger re-renders.
 *
 * @remarks
 * This hook stores scope in a ref, not state. The axios adapter reads the value at request time.
 * **Do not use this hook if you need to render based on the scope value.**
 */
export function useNamespaceScope(namespace: string | undefined) {
    const { setScope } = useScopeContext();

    useEffect(() => {
        if (namespace && namespace !== ALL_NAMESPACES_KEY) {
            setScope({ namespace });
        } else {
            setScope({});
        }
    }, [namespace, setScope]);
}

/**
 * Sets namespace and workload scope for all API requests. Side-effect only - does not trigger re-renders.
 *
 * @remarks
 * This hook stores scope in a ref, not state. The axios adapter reads the value at request time.
 * **Do not use this hook if you need to render based on the scope value.**
 *
 */
export function useWorkloadScope(namespace: string | undefined, workload: string | undefined) {
    const { setScope } = useScopeContext();

    useEffect(() => {
        if (namespace && namespace !== ALL_NAMESPACES_KEY && workload) {
            setScope({ namespace, workload });
        } else if (namespace && namespace !== ALL_NAMESPACES_KEY) {
            setScope({ namespace });
        } else {
            setScope({});
        }
    }, [namespace, workload, setScope]);
}
