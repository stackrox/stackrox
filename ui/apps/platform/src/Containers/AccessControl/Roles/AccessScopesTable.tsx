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
        <TableComposable variant="compact" isStickyHeader>
            <Thead>
                <Tr>
                    <Th key="radio" />
                    <Th key="name">Name</Th>
                    <Th key="description">Description</Th>
                </Tr>
            </Thead>
            <Tbody>
                <Tr className="pf-u-background-color-200">
                    <Td key="radio" className="pf-c-table__check">
                        <input
                            type="radio"
                            name={fieldId}
                            value=""
                            onChange={handleChange}
                            aria-label="No access scope"
                            checked={accessScopeId.length === 0}
                            disabled={isDisabled}
                        />
                    </Td>
                    <Td key="name" dataLabel="Name" modifier="nowrap">
                        No access scope
                    </Td>
                    <Td key="description" dataLabel="Description">
                        Role does not have an access scope
                    </Td>
                </Tr>
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
