import React, { ReactElement, useState } from 'react';
import {
    Alert,
    AlertVariant,
    Badge,
    Button,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { getIsDefaultRoleName } from 'constants/accessControl';
import { Group } from 'services/AuthService';
import { AccessScope, PermissionSet, Role } from 'services/RolesService';

import { AccessControlEntityLink } from '../AccessControlLinks';

// Return whether an auth provider rule refers to a role name,
// therefore need to disable the delete action for the role.
function getHasRoleName(groups: Group[], name: string) {
    return groups.some(({ roleName }) => roleName === name);
}

const entityType = 'ROLE';

export type RolesListProps = {
    roles: Role[];
    groups: Group[];
    permissionSets: PermissionSet[];
    accessScopes: AccessScope[];
    handleCreate: () => void;
    handleDelete: (id: string) => Promise<void>;
};

function RolesList({
    roles,
    groups,
    permissionSets,
    accessScopes,
    handleCreate,
    handleDelete,
}: RolesListProps): ReactElement {
    const [nameDeleting, setNameDeleting] = useState('');
    const [alertDelete, setAlertDelete] = useState<ReactElement | null>(null);

    function onClickDelete(name: string) {
        setNameDeleting(name);
        setAlertDelete(null);
        handleDelete(name)
            .catch((error) => {
                setAlertDelete(
                    <Alert title="Delete role failed" variant={AlertVariant.danger} isInline>
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
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
        <>
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarGroup spaceItems={{ default: 'spaceItemsMd' }}>
                        <ToolbarItem>
                            <Title headingLevel="h2">Roles</Title>
                        </ToolbarItem>
                        <ToolbarItem>
                            <Badge isRead>{roles.length}</Badge>
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarItem alignment={{ default: 'alignRight' }}>
                        <Button variant="primary" onClick={handleCreate} isSmall>
                            Add role
                        </Button>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            {alertDelete}
            {roles.length !== 0 && (
                <TableComposable variant="compact" isStickyHeader>
                    <Thead>
                        <Tr>
                            <Th width={20}>Name</Th>
                            <Th width={30}>Description</Th>
                            <Th width={20}>Permission set</Th>
                            <Th width={20}>Access scope</Th>
                            <Th width={10} aria-label="Row actions" />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {roles.map(({ name, description, permissionSetId, accessScopeId }) => (
                            <Tr key={name}>
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
                                        'No access scope'
                                    )}
                                </Td>
                                {getIsDefaultRoleName(name) || getHasRoleName(groups, name) ? (
                                    <Td />
                                ) : (
                                    <Td
                                        actions={{
                                            disable: nameDeleting === name,
                                            items: [
                                                {
                                                    title: 'Delete role',
                                                    onClick: () => onClickDelete(name),
                                                },
                                            ],
                                        }}
                                        className="pf-u-text-align-right"
                                    />
                                )}
                            </Tr>
                        ))}
                    </Tbody>
                </TableComposable>
            )}
        </>
    );
}

export default RolesList;
