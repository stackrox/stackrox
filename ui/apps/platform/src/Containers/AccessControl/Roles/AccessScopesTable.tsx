import React, { ChangeEventHandler, ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { AccessScope } from 'services/RolesService';

export type AccessScopesTableProps = {
    fieldId: string;
    accessScopeId: string;
    accessScopes: AccessScope[];
    handleChange: ChangeEventHandler<HTMLInputElement>;
    isDisabled: boolean;
};

function AccessScopesTable({
    fieldId,
    accessScopeId,
    accessScopes,
    handleChange,
    isDisabled,
}: AccessScopesTableProps): ReactElement {
    return (
        <TableComposable aria-label="Access scopes" variant="compact">
            <Thead>
                <Tr>
                    <Th key="radio" />
                    <Th key="name">Name</Th>
                    <Th key="description">Description</Th>
                </Tr>
            </Thead>
            <Tbody>
                {accessScopes.map(({ id, name, description }) => (
                    <Tr key={id}>
                        <Td key="radio" className="pf-c-table__check">
                            <input
                                type="radio"
                                name={fieldId}
                                value={id}
                                onChange={handleChange}
                                aria-label={name}
                                checked={id === accessScopeId}
                                disabled={isDisabled}
                            />
                        </Td>
                        <Td key="name" dataLabel="Name" modifier="nowrap">
                            {name}
                        </Td>
                        <Td key="description" dataLabel="Description">
                            {description}
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default AccessScopesTable;
