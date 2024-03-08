import React, { ChangeEventHandler, ReactElement } from 'react';
import { Table /* data-codemods */, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

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
        <Table variant="compact" isStickyHeader>
            <Thead>
                <Tr>
                    <Th key="radio" />
                    <Th width={20}>Name</Th>
                    <Th>Description</Th>
                </Tr>
            </Thead>
            <Tbody>
                {permissionSets.map(({ id, name, description }) => (
                    <Tr key={id}>
                        <Td className="pf-c-table__check">
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

export default PermissionSetsTable;
