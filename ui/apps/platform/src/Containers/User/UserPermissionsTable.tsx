import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { AccessLevel } from 'services/RolesService';

import {
    ReadAccessIcon,
    WriteAccessIcon,
} from 'Containers/AccessControl/PermissionSets/AccessIcons';

export type UserPermissionsTableProps = {
    permissions: Record<string, AccessLevel>;
};

function UserPermissionsTable({ permissions }: UserPermissionsTableProps): ReactElement {
    return (
        <TableComposable aria-label="Permissions" variant="compact">
            <Thead>
                <Tr>
                    <Th key="resourceName">Resource</Th>
                    <Th key="read">Read</Th>
                    <Th key="write">Write</Th>
                </Tr>
            </Thead>
            <Tbody>
                {Object.entries(permissions).map(([resource, accessType]) => (
                    <Tr key={resource}>
                        <Td key="resourceName" dataLabel="Resource">
                            {resource}
                        </Td>
                        <Td key="read" dataLabel="Read" data-testid="read">
                            <ReadAccessIcon accessType={accessType} />
                        </Td>
                        <Td key="write" dataLabel="Write" data-testid="write">
                            <WriteAccessIcon accessType={accessType} />
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default UserPermissionsTable;
