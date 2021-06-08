import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { AccessControlEntityLink } from '../AccessControlLinks';
import { AuthProvider } from '../accessControlTypes';

// TODO import from where?
const unselectedRowStyle = {};
const selectedRowStyle = {
    borderLeft: '3px solid var(--pf-global--primary-color--100)',
};

const entityType = 'AUTH_PROVIDER';

export type AuthProvidersListProps = {
    entityId?: string;
    authProviders: AuthProvider[];
};

function AuthProvidersList({ entityId, authProviders }: AuthProvidersListProps): ReactElement {
    return (
        <TableComposable variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Auth provider</Th>
                    <Th>Minimum access role</Th>
                </Tr>
            </Thead>
            <Tbody>
                {authProviders.map(({ id, name, authProvider, minimumAccessRole }) => (
                    <Tr key={id} style={id === entityId ? selectedRowStyle : unselectedRowStyle}>
                        <Td dataLabel="Name">
                            <AccessControlEntityLink
                                entityType={entityType}
                                entityId={id}
                                entityName={name}
                            />
                        </Td>
                        <Td dataLabel="Auth provider">{authProvider}</Td>
                        <Td dataLabel="Minimum access role">
                            <AccessControlEntityLink
                                entityType="ROLE"
                                entityId={minimumAccessRole}
                                entityName={minimumAccessRole}
                            />
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default AuthProvidersList;
