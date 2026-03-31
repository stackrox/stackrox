import type { ReactElement } from 'react';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import RolesForResourceAccess from './RolesForResourceAccess';

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
        <Table aria-label="Permissions for roles" variant="compact">
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
        </Table>
    );
}

export default UserPermissionsForRolesTable;
