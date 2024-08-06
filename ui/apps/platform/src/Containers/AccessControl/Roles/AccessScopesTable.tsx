import React, { ChangeEventHandler, ReactElement } from 'react';
import { Table, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

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
        <Table variant="compact" isStickyHeader>
            <Thead>
                <Tr>
                    <Th>
                        <span className="pf-v5-screen-reader">Row selection</span>
                    </Th>
                    <Th width={20}>Name</Th>
                    <Th>Description</Th>
                </Tr>
            </Thead>
            <Tbody>
                {accessScopes.map(({ id, name, description }) => (
                    <Tr key={id}>
                        <Td className="pf-v5-c-table__check">
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
        </Table>
    );
}

export default AccessScopesTable;
