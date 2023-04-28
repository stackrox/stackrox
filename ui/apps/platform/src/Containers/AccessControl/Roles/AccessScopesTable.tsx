import React, { ChangeEventHandler, ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { AccessScope } from 'services/AccessScopesService';

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
                    <Td />
                    <Th width={20}>Name</Th>
                    <Th>Description</Th>
                </Tr>
            </Thead>
            <Tbody>
                {accessScopes.map(({ id, name, description }) => (
                    <Tr key={id}>
                        <Td className="pf-c-table__check">
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
                        <Td dataLabel="Name" modifier="nowrap">
                            {name}
                        </Td>
                        <Td dataLabel="Description">{description}</Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default AccessScopesTable;
