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

import { PermissionSet, Role } from 'services/RolesService';

import { AccessControlEntityLink, RolesLink } from '../AccessControlLinks';
import usePermissions from '../../../hooks/usePermissions';
import { getOriginLabel, isUserResource } from '../traits';

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
    handleDelete,
}: PermissionSetsListProps): ReactElement {
    const [idDeleting, setIdDeleting] = useState('');
    const [nameConfirmingDelete, setNameConfirmingDelete] = useState<string | null>(null);
    const [alertDelete, setAlertDelete] = useState<ReactElement | null>(null);
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPage = hasReadWriteAccess('Access');

    function onClickDelete(id: string) {
        setIdDeleting(id);
        setNameConfirmingDelete(
            permissionSets.find((permissionSet) => permissionSet.id === id)?.name ?? ''
        );
    }

    function onConfirmDelete() {
        setNameConfirmingDelete(null);
        setAlertDelete(null);
        handleDelete(idDeleting)
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

    function onCancelDelete() {
        setNameConfirmingDelete(null);
        setIdDeleting('');
    }

    return (
        <PageSection variant="light">
            <Title headingLevel="h2">{pluralize(permissionSets.length, 'result')} found</Title>
            {alertDelete}
            {permissionSets.length !== 0 && (
                <TableComposable variant="compact" isStickyHeader>
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
                        {permissionSets.map(({ id, name, description, traits }) => (
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
                                            ({ permissionSetId }) => permissionSetId === id
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
            <Modal
                variant={ModalVariant.small}
                title="Permanently delete permission set?"
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
                        Permission set name: <strong>{nameConfirmingDelete}</strong>
                    </div>
                ) : (
                    ''
                )}
            </Modal>
        </PageSection>
    );
}

export default PermissionSetsList;
