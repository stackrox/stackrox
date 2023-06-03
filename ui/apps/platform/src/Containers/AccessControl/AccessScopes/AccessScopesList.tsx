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
import { Role } from 'services/RolesService';

import { AccessControlEntityLink, RolesLink } from '../AccessControlLinks';
import usePermissions from '../../../hooks/usePermissions';
import { getOriginLabel, isUserResource } from '../traits';

const entityType = 'ACCESS_SCOPE';

export type AccessScopesListProps = {
    accessScopes: AccessScope[];
    roles: Role[];
    handleDelete: (id: string) => Promise<void>;
};

function AccessScopesList({
    accessScopes,
    roles,
    handleDelete,
}: AccessScopesListProps): ReactElement {
    const [idDeleting, setIdDeleting] = useState('');
    const [nameConfirmingDelete, setNameConfirmingDelete] = useState<string | null>(null);
    const [alertDelete, setAlertDelete] = useState<ReactElement | null>(null);
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPage = hasReadWriteAccess('Access');

    function onClickDelete(id: string) {
        setIdDeleting(id);
        setNameConfirmingDelete(
            accessScopes.find((accessScope) => accessScope.id === id)?.name ?? ''
        );
    }

    function onConfirmDelete() {
        setNameConfirmingDelete(null);
        setAlertDelete(null);
        handleDelete(idDeleting)
            .catch((error) => {
                setAlertDelete(
                    <Alert
                        title="Delete access scope failed"
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

    function onCancelDelete() {
        setNameConfirmingDelete(null);
        setIdDeleting('');
    }

    return (
        <PageSection variant="light">
            <Title headingLevel="h2">{pluralize(accessScopes.length, 'result')} found</Title>
            {alertDelete}
            <TableComposable variant="compact">
                <Thead>
                    <Tr>
                        <Th width={15}>Name</Th>
                        <Th width={15}>Origin</Th>
                        <Th width={25}>Description</Th>
                        <Th width={35}>Roles</Th>
                        <Th width={10} aria-label="Row actions" />
                    </Tr>
                </Thead>
                <Tbody>
                    {accessScopes.map(({ id, name, description, traits }) => (
                        <Tr key={id}>
                            <Td dataLabel="Name">
                                <AccessControlEntityLink
                                    entityType={entityType}
                                    entityId={id}
                                    entityName={name}
                                />
                            </Td>
                            <Td dataLabel="Origin">{getOriginLabel(traits)}</Td>
                            <Td dataLabel="Description">{description}</Td>
                            <Td dataLabel="Roles">
                                <RolesLink
                                    roles={roles.filter(
                                        ({ accessScopeId }) => accessScopeId === id
                                    )}
                                    entityType={entityType}
                                    entityId={id}
                                />
                            </Td>
                            <Td
                                actions={{
                                    disable:
                                        !hasWriteAccessForPage ||
                                        idDeleting === id ||
                                        !isUserResource(traits) ||
                                        roles.some(({ accessScopeId }) => accessScopeId === id),
                                    items: [
                                        {
                                            title: 'Delete access scope',
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
            <Modal
                variant={ModalVariant.small}
                title="Permanently delete access scope?"
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
                        Access scope name: <strong>{nameConfirmingDelete}</strong>
                    </div>
                ) : (
                    ''
                )}
            </Modal>
        </PageSection>
    );
}

export default AccessScopesList;
