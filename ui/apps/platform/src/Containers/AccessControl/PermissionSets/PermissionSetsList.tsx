import React, { ReactElement, useState } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { accessControl as accessControlTypeLabels } from 'messages/common';
import { PermissionSet, Role } from 'services/RolesService';

import { AccessControlEntityLink, RolesLink } from '../AccessControlLinks';

// TODO import from where?
const unselectedRowStyle = {};
const selectedRowStyle = {
    borderLeft: '3px solid var(--pf-global--primary-color--100)',
};

const entityType = 'PERMISSION_SET';

export type PermissionSetsListProps = {
    entityId?: string;
    permissionSets: PermissionSet[];
    roles: Role[];
    handleDelete: (id: string) => Promise<void>;
};

function PermissionSetsList({
    entityId,
    permissionSets,
    roles,
    handleDelete,
}: PermissionSetsListProps): ReactElement {
    const [idDeleting, setIdDeleting] = useState('');

    function onClickDelete(id: string) {
        setIdDeleting(id);
        handleDelete(id).finally(() => {
            setIdDeleting('');
        });
    }

    return (
        <TableComposable variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Description</Th>
                    <Th>Minimum access level</Th>
                    <Th>Roles</Th>
                    <Th aria-label="Row actions" />
                </Tr>
            </Thead>
            <Tbody>
                {permissionSets.map(({ id, name, description, minimumAccessLevel }) => (
                    <Tr key={id} style={id === entityId ? selectedRowStyle : unselectedRowStyle}>
                        <Td dataLabel="Name">
                            <AccessControlEntityLink
                                entityType={entityType}
                                entityId={id}
                                entityName={name}
                            />
                        </Td>
                        <Td dataLabel="Description">{description}</Td>
                        <Td dataLabel="Minimum access level">
                            {accessControlTypeLabels[minimumAccessLevel] ?? ''}
                        </Td>
                        <Td dataLabel="Roles">
                            <RolesLink
                                roles={roles.filter(
                                    ({ permissionSetId }) => permissionSetId === id
                                )}
                                entityType={entityType}
                                entityId={id}
                            />
                        </Td>
                        {roles.some(({ permissionSetId }) => permissionSetId === id) ? (
                            <Td />
                        ) : (
                            <Td
                                actions={{
                                    disable: Boolean(entityId) || idDeleting === id,
                                    items: [
                                        {
                                            title: 'Delete permission set',
                                            onClick: () => onClickDelete(id),
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

export default PermissionSetsList;
