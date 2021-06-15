import React, { ReactElement, useState } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { AccessScope, Role } from 'services/RolesService';

import { AccessControlEntityLink, RolesLink } from '../AccessControlLinks';

// TODO import from where?
const unselectedRowStyle = {};
const selectedRowStyle = {
    borderLeft: '3px solid var(--pf-global--primary-color--100)',
};

const entityType = 'ACCESS_SCOPE';

export type AccessScopesListProps = {
    entityId?: string;
    accessScopes: AccessScope[];
    roles: Role[];
    handleDelete: (id: string) => Promise<void>;
};

function AccessScopesList({
    entityId,
    accessScopes,
    roles,
    handleDelete,
}: AccessScopesListProps): ReactElement {
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
                    <Th>Roles</Th>
                    <Th aria-label="Row actions" />
                </Tr>
            </Thead>
            <Tbody>
                {accessScopes.map(({ id, name, description }) => (
                    <Tr key={id} style={id === entityId ? selectedRowStyle : unselectedRowStyle}>
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
                                roles={roles.filter(({ accessScopeId }) => accessScopeId === id)}
                                entityType={entityType}
                                entityId={id}
                            />
                        </Td>
                        {roles.some(({ accessScopeId }) => accessScopeId === id) ? (
                            <Td />
                        ) : (
                            <Td
                                actions={{
                                    disable: Boolean(entityId) || idDeleting === id,
                                    items: [
                                        {
                                            title: 'Delete access scope',
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

export default AccessScopesList;
