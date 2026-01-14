import { useEffect, useState } from 'react';

import { ScopeProvider, useNamespaceScope, useScopeContext } from './ScopeContext';

function ScopeTest({ namespace }) {
    const { getScope } = useScopeContext();
    const [scope, setScope] = useState(getScope());

    useNamespaceScope(namespace);

    // NOTE: This useState + useEffect pattern is ONLY for testing purposes to display scope values.
    // In production code, you should NOT read scope for rendering - it's stored in a ref and doesn't
    // trigger re-renders. Real components just call useNamespaceScope to SET scope,
    // and the axios adapter reads it at request time.
    useEffect(() => {
        setScope(getScope());
    }, [namespace, getScope]);

    return (
        <div>
            <div data-testid="namespace">{scope.namespace || 'none'}</div>
        </div>
    );
}

describe(Cypress.spec.relative, () => {
    it('should update scope correctly through lifecycle transitions', () => {
        function TestWrapper() {
            const [namespace, setNamespace] = useState(undefined);

            return (
                <ScopeProvider>
                    <ScopeTest namespace={namespace} />
                    <button type="button" onClick={() => setNamespace('test-ns')}>
                        Set Namespace
                    </button>
                    <button type="button" onClick={() => setNamespace(undefined)}>
                        Clear Namespace
                    </button>
                </ScopeProvider>
            );
        }

        cy.mount(<TestWrapper />);

        // Initial: no scope
        cy.findByTestId('namespace').should('have.text', 'none');

        // Transition 1: namespace only
        cy.findByText('Set Namespace').click();
        cy.findByTestId('namespace').should('have.text', 'test-ns');

        // Transition 2: back to no scope
        cy.findByText('Clear Namespace').click();
        cy.findByTestId('namespace').should('have.text', 'none');
    });
});
