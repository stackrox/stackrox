import React, { CSSProperties, ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import RolesForResourceAccess from './RolesForResourceAccess';

// Normal horizontal padding to separate icons from text in the preceding cell.
const style = {
    '--pf-c-table--m-compact--cell--PaddingLeft': 'var(--pf-global--spacer--md)',
    '--pf-c-table--m-compact--cell--PaddingRight': 'var(--pf-global--spacer--md)',
} as CSSProperties;

export type PermissionByRole = {
    read: string[];
    write: string[];
};

export type UserPermissionsForRolesTableProps = {
    resourceToAccessByRole: Record<string, PermissionByRole>;
};

function UserPermissionsForRolesTable({
    resourceToAccessByRole,
}: UserPermissionsForRolesTableProps): ReactElement {
    return (
        <TableComposable aria-label="Permissions for roles" variant="compact" style={style}>
            <Thead>
                <Tr>
                    <Th key="resourceName">Resource</Th>
                    <Th key="read">Read</Th>
                    <Th key="write">Write</Th>
                </Tr>
            </Thead>
            <Tbody>
                {Object.entries(resourceToAccessByRole).map(([resource, { read, write }]) => (
                    <Tr key={resource}>
                        <Td key="resourceName" dataLabel="Resource">
                            {resource}
                        </Td>
                        <Td key="read" dataLabel="Read" data-testid="read">
                            <RolesForResourceAccess roleNames={read} />
                        </Td>
                        <Td key="write" dataLabel="Write" data-testid="write">
                            <RolesForResourceAccess roleNames={write} />
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default UserPermissionsForRolesTable;
