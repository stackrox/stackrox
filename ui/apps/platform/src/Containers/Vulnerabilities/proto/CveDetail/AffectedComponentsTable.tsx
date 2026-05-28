import { Bullseye, EmptyState, EmptyStateBody } from '@patternfly/react-core';

/**
 * Placeholder for affected components table.
 * Component data is not yet available in the prototype GraphQL schema.
 */
function AffectedComponentsTable() {
    return (
        <Bullseye>
            <EmptyState>
                <EmptyStateBody>
                    Components data coming soon — not yet wired to the prototype GraphQL schema.
                </EmptyStateBody>
            </EmptyState>
        </Bullseye>
    );
}

export default AffectedComponentsTable;
