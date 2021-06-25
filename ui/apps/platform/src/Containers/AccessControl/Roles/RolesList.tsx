import React, { ReactElement, useState } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { defaultRoles } from 'constants/accessControl';
import { AccessScope, PermissionSet, Role } from 'services/RolesService';

import { AccessControlEntityLink } from '../AccessControlLinks';

// TODO import from where?
const unselectedRowStyle = {};
const selectedRowStyle = {
    borderLeft: '3px solid var(--pf-global--primary-color--100)',
};

const entityType = 'ROLE';

export type RolesListProps = {
    entityName?: string;
    roles: Role[];
    permissionSets: PermissionSet[];
    accessScopes: AccessScope[];
    handleDelete: (id: string) => Promise<void>;
};

function RolesList({
    entityName,
    roles,
    permissionSets,
    accessScopes,
    handleDelete,
}: RolesListProps): ReactElement {
    const [nameDeleting, setNameDeleting] = useState('');

    function onClickDelete(name: string) {
        setNameDeleting(name);
        handleDelete(name).finally(() => {
            setNameDeleting('');
        });
    }

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
                    <Th aria-label="Row actions" />
                </Tr>
            </Thead>
            <Tbody>
                {roles.map(({ name, description, permissionSetId, accessScopeId }) => (
                    <Tr
                        key={name}
                        style={name === entityName ? selectedRowStyle : unselectedRowStyle}
                    >
                        <Td dataLabel="Name">
                            <AccessControlEntityLink
                                entityType={entityType}
                                entityId={name}
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
                            {accessScopeId ? (
                                <AccessControlEntityLink
                                    entityType="ACCESS_SCOPE"
                                    entityId={accessScopeId}
                                    entityName={getAccessScopeName(accessScopeId)}
                                />
                            ) : (
                                'Unlimited access'
                            )}
                        </Td>
                        {defaultRoles[name] ? (
                            <Td />
                        ) : (
                            <Td
                                actions={{
                                    disable: Boolean(entityName) || nameDeleting === name,
                                    items: [
                                        {
                                            title: 'Delete role',
                                            onClick: () => onClickDelete(name),
                                        },
                                    ],
                                }}
                            />
                        )}
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default RolesList;
