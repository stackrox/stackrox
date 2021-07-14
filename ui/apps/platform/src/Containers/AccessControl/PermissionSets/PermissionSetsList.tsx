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

import { PermissionSet, Role } from 'services/RolesService';

import { AccessControlEntityLink, RolesLink } from '../AccessControlLinks';

const entityType = 'PERMISSION_SET';

export type PermissionSetsListProps = {
    permissionSets: PermissionSet[];
    roles: Role[];
    handleCreate: () => void;
    handleDelete: (id: string) => Promise<void>;
};

function PermissionSetsList({
    permissionSets,
    roles,
    handleCreate,
    handleDelete,
}: PermissionSetsListProps): ReactElement {
    const [idDeleting, setIdDeleting] = useState('');
    const [alertDelete, setAlertDelete] = useState<ReactElement | null>(null);

    function onClickDelete(id: string) {
        setIdDeleting(id);
        setAlertDelete(null);
        handleDelete(id)
            .catch((error) => {
                setAlertDelete(
                    <Alert
                        title="Delete permission set failed"
                        variant={AlertVariant.danger}
                        isInline
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setIdDeleting('');
            });
    }

    return (
        <>
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarGroup spaceItems={{ default: 'spaceItemsMd' }}>
                        <ToolbarItem>
                            <Title headingLevel="h2">Permission sets</Title>
                        </ToolbarItem>
                        <ToolbarItem>
                            <Badge isRead>{permissionSets.length}</Badge>
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarItem alignment={{ default: 'alignRight' }}>
                        <Button variant="primary" onClick={handleCreate} isSmall>
                            Add permission set
                        </Button>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            {alertDelete}
            {permissionSets.length !== 0 && (
                <TableComposable variant="compact" isStickyHeader>
                    <Thead>
                        <Tr>
                            <Th width={20}>Name</Th>
                            <Th width={30}>Description</Th>
                            <Th width={40}>Roles</Th>
                            <Th width={10} aria-label="Row actions" />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {permissionSets.map(({ id, name, description }) => (
                            <Tr key={id}>
                                <Td dataLabel="Name">
                                    <AccessControlEntityLink
                                        entityType={entityType}
                                        entityId={id}
                                        entityName={name}
                                    />
                                </Td>
                                <Td dataLabel="Description">{description}</Td>
                                <Td dataLabel="Roles">
                                    <RolesLink
                                        roles={roles.filter(
                                            ({ permissionSetId }) => permissionSetId === id
                                        )}
                                        entityType={entityType}
                                        entityId={id}
                                    />
                                </Td>
                                <Td
                                    actions={{
                                        disable:
                                            idDeleting === id ||
                                            roles.some(
                                                ({ permissionSetId }) => permissionSetId === id
                                            ),
                                        items: [
                                            {
                                                title: 'Delete permission set',
                                                onClick: () => onClickDelete(id),
                                            },
                                        ],
                                    }}
                                    className="pf-u-text-align-right"
                                />
                            </Tr>
                        ))}
                    </Tbody>
                </TableComposable>
            )}
        </>
    );
}

export default PermissionSetsList;
