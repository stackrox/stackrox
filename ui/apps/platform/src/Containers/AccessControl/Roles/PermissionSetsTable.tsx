import React, { ChangeEventHandler, ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { PermissionSet } from 'services/RolesService';

export type PermissionSetsTableProps = {
    fieldId: string;
    permissionSetId: string;
    permissionSets: PermissionSet[];
    handleChange: ChangeEventHandler<HTMLInputElement>;
    isDisabled: boolean;
};

function PermissionSetsTable({
    fieldId,
    permissionSetId,
    permissionSets,
    handleChange,
    isDisabled,
}: PermissionSetsTableProps): ReactElement {
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
                {permissionSets.map(({ id, name, description }) => (
                    <Tr key={id}>
                        <Td key="radio" className="pf-c-table__check">
                            <input
                                type="radio"
                                name={fieldId}
                                value={id}
                                onChange={handleChange}
                                aria-label={name}
                                checked={id === permissionSetId}
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

export default PermissionSetsTable;
