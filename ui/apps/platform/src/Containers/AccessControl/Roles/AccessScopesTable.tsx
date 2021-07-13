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
                    <Th />
                    <Th width={20}>Name</Th>
                    <Th>Description</Th>
                </Tr>
            </Thead>
            <Tbody>
                <Tr className="pf-u-background-color-200">
                    <Td className="pf-c-table__check">
                        <input
                            type="radio"
                            name={fieldId}
                            value=""
                            onChange={handleChange}
                            aria-label="Unrestricted"
                            checked={accessScopeId.length === 0}
                            disabled={isDisabled}
                        />
                    </Td>
                    <Td dataLabel="Name" modifier="nowrap">
                        Unrestricted
                    </Td>
                    <Td dataLabel="Description">Access to all clusters and namespaces</Td>
                </Tr>
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
