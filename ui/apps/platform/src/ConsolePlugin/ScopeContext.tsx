import { createContext, useContext, useRef } from 'react';
import type { MutableRefObject, ReactNode } from 'react';
import { useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';

import type { Empty } from 'services/types';

import { ALL_NAMESPACES_KEY } from './constants';

export type AuthScope = Empty | { namespace: string };

const ScopeContext = createContext<MutableRefObject<AuthScope>>({ current: {} });

/**
 * Provider that automatically tracks the active namespace and updates scope.
 * Provides a ref object via context - the ref's .current value is read by axios at request time.
 */
export function ScopeProvider({ children }: { children: ReactNode }) {
    const [namespace] = useActiveNamespace();
    const scopeRef = useRef<AuthScope>({});

    if (namespace && namespace !== ALL_NAMESPACES_KEY) {
        scopeRef.current = { namespace };
    } else {
        scopeRef.current = {};
    }

    return <ScopeContext.Provider value={scopeRef}>{children}</ScopeContext.Provider>;
}

/**
 * Hook to access the scope ref.
 * Returns a ref object whose .current property contains the current auth scope.
 */
export function useScope() {
    return useContext(ScopeContext);
}

/**
 * Sets namespace scope for all API requests. Side-effect only - does not trigger re-renders.
 *
 * @remarks
 * Use this to override the automatic namespace tracking from ScopeProvider.
 * Updates the scope ref directly during render to ensure scope is set before any child effects run.
 * **Do not use this hook if you need to render based on the scope value.**
 */
export function useNamespaceScope(namespace: string | undefined) {
    const scopeRef = useScope();

    if (namespace && namespace !== ALL_NAMESPACES_KEY) {
        scopeRef.current = { namespace };
    } else {
        scopeRef.current = {};
    }
}
