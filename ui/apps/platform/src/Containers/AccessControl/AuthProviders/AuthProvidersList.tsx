import React, { useState, ReactElement } from 'react';
import pluralize from 'pluralize';
import { useSelector, useDispatch } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Button, Modal, ModalVariant } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';
import { AuthProvider, AuthProviderInfo, getIsAuthProviderImmutable } from 'services/AuthService';

import { AccessControlEntityLink } from '../AccessControlLinks';
import { getOriginLabel } from '../traits';

// TODO import from where?
const unselectedRowStyle = {};
const selectedRowStyle = {
    borderLeft: '3px solid var(--pf-global--primary-color--100)',
};

function getAuthProviderTypeLabel(type: string, availableTypes: AuthProviderInfo[]): string {
    return availableTypes.find(({ value }) => value === type)?.label ?? '';
}

const entityType = 'AUTH_PROVIDER';

export type AuthProvidersListProps = {
    entityId?: string;
    authProviders: AuthProvider[];
};

const authProviderState = createStructuredSelector({
    currentUser: selectors.getCurrentUser,
    availableProviderTypes: selectors.getAvailableProviderTypes,
});

function AuthProvidersList({ entityId, authProviders }: AuthProvidersListProps): ReactElement {
    const [authProviderToDelete, setAuthProviderToDelete] = useState('');
    const [idToDelete, setIdToDelete] = useState('');
    const dispatch = useDispatch();
    const { currentUser, availableProviderTypes } = useSelector(authProviderState);

    function onClickDelete(name: string, id: string) {
        setIdToDelete(id);
        setAuthProviderToDelete(name);
    }

    function confirmDelete() {
        dispatch(authActions.deleteAuthProvider(idToDelete));
        clearPendingDelete();
    }

    function clearPendingDelete() {
        setIdToDelete('');
        setAuthProviderToDelete('');
    }

    return (
        <>
            <TableComposable variant="compact">
                <Thead>
                    <Tr>
                        <Th width={15}>Name</Th>
                        <Th width={15}>Origin</Th>
                        <Th width={15}>Type</Th>
                        <Th width={20}>Minimum access role</Th>
                        <Th width={25}>Assigned rules</Th>
                        <Th width={10} aria-label="Row actions" />
                    </Tr>
                </Thead>
                <Tbody>
                    {authProviders.map((authProvider) => {
                        const { id, name, type, defaultRole, traits, groups = [] } = authProvider;
                        const typeLabel = getAuthProviderTypeLabel(type, availableProviderTypes);
                        const isImmutable = getIsAuthProviderImmutable(authProvider);

                        return (
                            <Tr
                                key={id}
                                style={id === entityId ? selectedRowStyle : unselectedRowStyle}
                            >
                                <Td dataLabel="Name">
                                    <AccessControlEntityLink
                                        entityType={entityType}
                                        entityId={id}
                                        entityName={name}
                                    />
                                </Td>
                                <Td dataLabel="Origin">{getOriginLabel(traits)}</Td>
                                <Td dataLabel="Type">{typeLabel}</Td>
                                <Td dataLabel="Minimum access role">
                                    <AccessControlEntityLink
                                        entityType="ROLE"
                                        entityId={defaultRole || ''}
                                        entityName={defaultRole || ''}
                                    />
                                </Td>
                                <Td dataLabel="Assigned rules">
                                    {`${groups.length} ${pluralize('rules', groups.length)}`}
                                </Td>
                                <Td
                                    actions={{
                                        items: [
                                            {
                                                title: 'Delete auth provider',
                                                onClick: () => onClickDelete(name, id),
                                                isDisabled:
                                                    id === currentUser?.authProvider?.id ||
                                                    isImmutable,
                                                description:
                                                    // eslint-disable-next-line no-nested-ternary
                                                    id === currentUser?.authProvider?.id
                                                        ? 'Cannot delete current auth provider'
                                                        : isImmutable
                                                          ? 'Cannot delete unmodifiable auth provider'
                                                          : '',
                                            },
                                        ],
                                    }}
                                    className="pf-u-text-align-right"
                                />
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
            <Modal
                variant={ModalVariant.small}
                title="Permanently delete auth provider?"
                isOpen={!!authProviderToDelete}
                onClose={clearPendingDelete}
                actions={[
                    <Button key="confirm" variant="danger" onClick={confirmDelete}>
                        Delete
                    </Button>,
                    <Button key="cancel" variant="link" onClick={clearPendingDelete}>
                        Cancel
                    </Button>,
                ]}
            >
                If you delete <span>{authProviderToDelete}</span>, no user of this auth provider
                will be able to use it to log in anymore.
            </Modal>
        </>
    );
}

export default AuthProvidersList;
