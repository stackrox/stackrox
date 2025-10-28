import { useState } from 'react';
import type { ReactElement } from 'react';
import { Alert, Button, Modal, PageSection, pluralize, Title } from '@patternfly/react-core';
import { ActionsColumn, Table, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import usePermissions from 'hooks/usePermissions';
import type { PermissionSet, Role } from 'services/RolesService';
import { getOriginLabel, isUserResource } from 'utils/traits.utils';

import { AccessControlEntityLink, RolesLink } from '../AccessControlLinks';

const entityType = 'PERMISSION_SET';

export type PermissionSetsListProps = {
    permissionSets: PermissionSet[];
    roles: Role[];
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
                        component="p"
                        variant="danger"
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
                <Table variant="compact" isStickyHeader>
                    <Thead>
                        <Tr>
                            <Th width={15}>Name</Th>
                            <Th width={15}>Origin</Th>
                            <Th width={25}>Description</Th>
                            <Th width={35}>Roles</Th>
                            <Th width={10}>
                                <span className="pf-v5-screen-reader">Row actions</span>
                            </Th>
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
                                <Td isActionCell>
                                    <ActionsColumn
                                        isDisabled={
                                            !hasWriteAccessForPage ||
                                            idDeleting === id ||
                                            !isUserResource(traits) ||
                                            roles.some(
                                                ({ permissionSetId }) => permissionSetId === id
                                            )
                                        }
                                        items={[
                                            {
                                                title: 'Delete permission set',
                                                onClick: () => onClickDelete(id),
                                            },
                                        ]}
                                    />
                                </Td>
                            </Tr>
                        ))}
                    </Tbody>
                </Table>
            )}
            <Modal
                variant="small"
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
