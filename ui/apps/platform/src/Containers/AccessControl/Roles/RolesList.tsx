import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { AccessControlEntityLink } from '../AccessControlLinks';
import { AccessScope, PermissionSet, Role } from '../accessControlTypes';

// TODO import from where?
const unselectedRowStyle = {};
const selectedRowStyle = {
    borderLeft: '3px solid var(--pf-global--primary-color--100)',
};

const entityType = 'ROLE';

export type RolesListProps = {
    entityId?: string;
    roles: Role[];
    permissionSets: PermissionSet[];
    accessScopes: AccessScope[];
};

function RolesList({
    entityId,
    roles,
    permissionSets,
    accessScopes,
}: RolesListProps): ReactElement {
    function getPermissionSetName(permissionSetId: string): string {
        return permissionSets.find(({ id }) => id === permissionSetId)?.name ?? '';
    }

    function getAccessScopeName(accessScopeId: string): string {
        return accessScopes.find(({ id }) => id === accessScopeId)?.name ?? '';
    }

    return (
        <TableComposable variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Description</Th>
                    <Th>Permission set</Th>
                    <Th>Access scope</Th>
                </Tr>
            </Thead>
            <Tbody>
                {roles.map(({ id, name, description, permissionSetId, accessScopeId }) => (
                    <Tr key={id} style={id === entityId ? selectedRowStyle : unselectedRowStyle}>
                        <Td dataLabel="Name">
                            <AccessControlEntityLink
                                entityType={entityType}
                                entityId={id}
                                entityName={name}
                            />
                        </Td>
                        <Td dataLabel="Description">{description}</Td>
                        <Td dataLabel="Permission set">
                            <AccessControlEntityLink
                                entityType="PERMISSION_SET"
                                entityId={permissionSetId}
                                entityName={getPermissionSetName(permissionSetId)}
                            />
                        </Td>
                        <Td dataLabel="Access scope">
                            <AccessControlEntityLink
                                entityType="ACCESS_SCOPE"
                                entityId={accessScopeId}
                                entityName={getAccessScopeName(accessScopeId)}
                            />
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default RolesList;
