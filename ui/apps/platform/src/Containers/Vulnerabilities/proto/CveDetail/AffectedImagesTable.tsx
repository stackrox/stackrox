import { Bullseye, EmptyState, EmptyStateBody } from '@patternfly/react-core';

/**
 * Placeholder for affected images table.
 * Image association data is not yet available in the prototype GraphQL schema.
 */
function AffectedImagesTable() {
    return (
        <Bullseye>
            <EmptyState>
                <EmptyStateBody>
                    Affected images data coming soon — not yet wired to the prototype GraphQL
                    schema.
                </EmptyStateBody>
            </EmptyState>
        </Bullseye>
    );
}

export default AffectedImagesTable;
