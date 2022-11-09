import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { PermissionsMap } from 'services/RolesService';

import {
    ReadAccessIcon,
    WriteAccessIcon,
} from 'Containers/AccessControl/PermissionSets/AccessIcons';
import {
    deprecatedResourceRowStyle,
    resourceRemovalReleaseVersions,
} from '../../constants/accessControl';
import { ResourceName } from '../../types/roleResources';

export type UserPermissionsTableProps = {
    permissions: PermissionsMap;
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
                {Object.entries(permissions).map(([resource, accessLevel]) => (
                    <Tr
                        key={resource}
                        style={
                            resourceRemovalReleaseVersions.has(resource as ResourceName)
                                ? deprecatedResourceRowStyle
                                : {}
                        }
                    >
                        <Td key="resourceName" dataLabel="Resource">
                            {resource}
                        </Td>
                        <Td key="read" dataLabel="Read" data-testid="read">
                            <ReadAccessIcon accessLevel={accessLevel} />
                        </Td>
                        <Td key="write" dataLabel="Write" data-testid="write">
                            <WriteAccessIcon accessLevel={accessLevel} />
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default UserPermissionsTable;
