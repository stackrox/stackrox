import { createContext, useContext, useRef } from 'react';
import type { MutableRefObject, ReactNode } from 'react';
import { useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';

import type { Empty } from 'services/types';

export type AuthScope = Empty | { namespace: string };

const ScopeContext = createContext<MutableRefObject<AuthScope>>({ current: {} });

/**
 * Provider that automatically tracks the active namespace and updates scope.
 * Provides a ref object via context - the ref's .current value is read by axios at request time.
 */
export function ScopeProvider({ children }: { children: ReactNode }) {
    const [namespace] = useActiveNamespace();
    const scopeRef = useRef<AuthScope>({});

    scopeRef.current = namespace ? { namespace } : {};

    return <ScopeContext.Provider value={scopeRef}>{children}</ScopeContext.Provider>;
}

/**
 * Hook to access the scope ref.
 * Returns a ref object whose .current property contains the current auth scope.
 */
export function useScope() {
    return useContext(ScopeContext);
}
