import React, { ReactElement, useState } from 'react';
import {
    Alert,
    AlertVariant,
    Button,
    Modal,
    ModalVariant,
    PageSection,
    pluralize,
    Title,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { AccessScope } from 'services/AccessScopesService';
import { Group } from 'services/AuthService';
import { PermissionSet, Role } from 'services/RolesService';

import { AccessControlEntityLink } from '../AccessControlLinks';
import { AccessControlQueryFilter } from '../accessControlPaths';
import usePermissions from '../../../hooks/usePermissions';
import { getOriginLabel, isUserResource } from '../traits';

// Return whether an auth provider rule refers to a role name,
// therefore need to disable the delete action for the role.
function getHasRoleName(groups: Group[], name: string) {
    return groups.some(({ roleName }) => roleName === name);
}

const entityType = 'ROLE';

export type RolesListProps = {
    roles: Role[];
    s?: AccessControlQueryFilter;
    groups: Group[];
    permissionSets: PermissionSet[];
    accessScopes: AccessScope[];
    handleDelete: (id: string) => Promise<void>;
};

function RolesList({
    roles,
    s,
    groups,
    permissionSets,
    accessScopes,
    handleDelete,
}: RolesListProps): ReactElement {
    const [nameDeleting, setNameDeleting] = useState('');
    const [nameConfirmingDelete, setNameConfirmingDelete] = useState<string | null>(null);
    const [alertDelete, setAlertDelete] = useState<ReactElement | null>(null);
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPage = hasReadWriteAccess('Access');

    function onClickDelete(name: string) {
        setNameDeleting(name);
        setNameConfirmingDelete(name);
    }

    function onConfirmDelete() {
        setNameConfirmingDelete(null);
        setAlertDelete(null);
        handleDelete(nameDeleting)
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

    function onCancelDelete() {
        setNameConfirmingDelete(null);
        setNameDeleting('');
    }

    function getPermissionSetName(permissionSetId: string): string {
        return permissionSets.find(({ id }) => id === permissionSetId)?.name ?? '';
    }

    function getAccessScopeName(accessScopeId: string): string {
        return accessScopes.find(({ id }) => id === accessScopeId)?.name ?? '';
    }

    const rolesFiltered = s
        ? roles.filter((role) => {
              if ('PERMISSION_SET' in s && role.permissionSetId !== s.PERMISSION_SET) {
                  return false;
              }
              if ('ACCESS_SCOPE' in s && role.accessScopeId !== s.ACCESS_SCOPE) {
                  return false;
              }
              return true;
          })
        : roles;

    return (
        <PageSection variant="light">
            <Title headingLevel="h2">{pluralize(rolesFiltered.length, 'result')} found</Title>
            {alertDelete}
            {rolesFiltered.length !== 0 && (
                <TableComposable variant="compact" isStickyHeader>
                    <Thead>
                        <Tr>
                            <Th width={15}>Name</Th>
                            <Th width={15}>Origin</Th>
                            <Th width={25}>Description</Th>
                            <Th width={15}>Permission set</Th>
                            <Th width={20}>Access scope</Th>
                            <Th width={10} aria-label="Row actions" />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {rolesFiltered.map(
                            ({ name, description, permissionSetId, accessScopeId, traits }) => (
                                <Tr key={name}>
                                    <Td dataLabel="Name">
                                        <AccessControlEntityLink
                                            entityType={entityType}
                                            entityId={name}
                                            entityName={name}
                                        />
                                    </Td>
                                    <Td dataLabel="Origin">{getOriginLabel(traits)}</Td>
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
                                    <Td
                                        actions={{
                                            disable:
                                                !hasWriteAccessForPage ||
                                                nameDeleting === name ||
                                                !isUserResource(traits) ||
                                                getHasRoleName(groups, name),
                                            items: [
                                                {
                                                    title: 'Delete role',
                                                    onClick: () => onClickDelete(name),
                                                },
                                            ],
                                        }}
                                        className="pf-u-text-align-right"
                                    />
                                </Tr>
                            )
                        )}
                    </Tbody>
                </TableComposable>
            )}
            <Modal
                variant={ModalVariant.small}
                title="Permanently delete role?"
                isOpen={typeof nameConfirmingDelete === 'string'}
                onClose={onCancelDelete}
                actions={[
                    <Button key="confirm" variant="danger" onClick={onConfirmDelete}>
                        Delete
                    </Button>,
                    <Button key="cancel" variant="link" onClick={onCancelDelete}>
                        Cancel
                    </Button>,
                ]}
            >
                {nameConfirmingDelete ? (
                    <div>
                        Role name: <strong>{nameConfirmingDelete}</strong>
                    </div>
                ) : (
                    ''
                )}
            </Modal>
        </PageSection>
    );
}

export default RolesList;
