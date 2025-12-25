import { useEffect, useState } from 'react';

import { ScopeProvider, useScopeContext, useWorkloadScope } from './ScopeContext';

function ScopeTest({ namespace, workload }) {
    const { getScope } = useScopeContext();
    const [scope, setScope] = useState(getScope());

    // useWorkloadScope handles all cases: namespace-only, namespace+workload, or empty
    useWorkloadScope(namespace, workload);

    // NOTE: This useState + useEffect pattern is ONLY for testing purposes to display scope values.
    // In production code, you should NOT read scope for rendering - it's stored in a ref and doesn't
    // trigger re-renders. Real components just call useNamespaceScope/useWorkloadScope to SET scope,
    // and the axios adapter reads it at request time.
    useEffect(() => {
        setScope(getScope());
    }, [namespace, workload, getScope]);

    return (
        <div>
            <div data-testid="namespace">{scope.namespace || 'none'}</div>
            <div data-testid="workload">{scope.workload || 'none'}</div>
        </div>
    );
}

describe(Cypress.spec.relative, () => {
    it('should update scope correctly through lifecycle transitions', () => {
        function TestWrapper() {
            const [namespace, setNamespace] = useState(undefined);
            const [workload, setWorkload] = useState(undefined);

            return (
                <ScopeProvider>
                    <ScopeTest namespace={namespace} workload={workload} />
                    <button type="button" onClick={() => setNamespace('test-ns')}>
                        Set Namespace
                    </button>
                    <button type="button" onClick={() => setWorkload('test-deploy')}>
                        Set Workload
                    </button>
                    <button type="button" onClick={() => setWorkload(undefined)}>
                        Clear Workload
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
        cy.findByTestId('workload').should('have.text', 'none');

        // Transition 1: namespace only
        cy.findByText('Set Namespace').click();
        cy.findByTestId('namespace').should('have.text', 'test-ns');
        cy.findByTestId('workload').should('have.text', 'none');

        // Transition 2: namespace + workload
        cy.findByText('Set Workload').click();
        cy.findByTestId('namespace').should('have.text', 'test-ns');
        cy.findByTestId('workload').should('have.text', 'test-deploy');

        // Transition 3: back to namespace only
        cy.findByText('Clear Workload').click();
        cy.findByTestId('namespace').should('have.text', 'test-ns');
        cy.findByTestId('workload').should('have.text', 'none');

        // Transition 4: back to no scope
        cy.findByText('Clear Namespace').click();
        cy.findByTestId('namespace').should('have.text', 'none');
        cy.findByTestId('workload').should('have.text', 'none');
    });
});
