import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { AccessControlEntityLink, RolesLink } from '../AccessControlLinks';
import { AccessScope, Role } from '../accessControlTypes';

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
};

function AccessScopesList({ entityId, accessScopes, roles }: AccessScopesListProps): ReactElement {
    return (
        <TableComposable variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Description</Th>
                    <Th>Roles</Th>
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
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default AccessScopesList;
