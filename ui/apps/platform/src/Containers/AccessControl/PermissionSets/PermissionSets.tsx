/* eslint-disable no-nested-ternary */
/* eslint-disable react/jsx-no-bind */
import React, { ReactElement, useEffect, useState } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertVariant,
    Badge,
    Bullseye,
    Button,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import { defaultRoles } from 'constants/accessControl';
import {
    PermissionSet,
    Role,
    createPermissionSet,
    deletePermissionSet,
    fetchPermissionSets,
    fetchResourcesAsArray,
    fetchRolesAsArray,
    updatePermissionSet,
} from 'services/RolesService';

import AccessControlNav from '../AccessControlNav';
import AccessControlPageTitle from '../AccessControlPageTitle';
import { getEntityPath, getQueryObject } from '../accessControlPaths';

import PermissionSetForm from './PermissionSetForm';
import PermissionSetsList from './PermissionSetsList';
import { getNewPermissionSet, getCompletePermissionSet } from './permissionSets.utils';

const entityType = 'PERMISSION_SET';

function PermissionSets(): ReactElement {
    const history = useHistory();
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;
    const { entityId } = useParams();

    const [isFetchingPrimary, setIsFetchingPrimary] = useState(false);
    const [permissionSets, setPermissionSets] = useState<PermissionSet[]>([]);
    const [alertPermissionSets, setAlertPermissionSets] = useState<ReactElement | null>(null);

    const [isFetchingSecondary, setIsFetchingSecondary] = useState(false);
    const [resources, setResources] = useState<string[]>([]);
    const [alertResources, setAlertResources] = useState<ReactElement | null>(null);

    const [roles, setRoles] = useState<Role[]>([]);
    const [alertRoles, setAlertRoles] = useState<ReactElement | null>(null);

    useEffect(() => {
        // The primary request has fetching spinner and unclosable alert.
        setIsFetchingPrimary(true);
        setAlertPermissionSets(null);
        fetchPermissionSets()
            .then((permissionSetsFetched) => {
                setPermissionSets(permissionSetsFetched);
            })
            .catch((error) => {
                setAlertPermissionSets(
                    <Alert
                        title="Fetch permission sets failed"
                        variant={AlertVariant.danger}
                        isInline
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setIsFetchingPrimary(false);
            });

        setIsFetchingSecondary(true);
        setAlertResources(null);
        fetchResourcesAsArray()
            .then((resourcesFetched) => {
                setResources(resourcesFetched);
            })
            .catch((error) => {
                const actionClose = <AlertActionCloseButton onClose={() => setAlertRoles(null)} />;
                setAlertRoles(
                    <Alert
                        title="Fetch resources failed"
                        variant={AlertVariant.warning}
                        isInline
                        actionClose={actionClose}
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setIsFetchingSecondary(false);
            });

        setAlertRoles(null);
        fetchRolesAsArray()
            .then((rolesFetched) => {
                setRoles(rolesFetched);
            })
            .catch((error) => {
                const actionClose = <AlertActionCloseButton onClose={() => setAlertRoles(null)} />;
                setAlertRoles(
                    <Alert
                        title="Fetch roles failed"
                        variant={AlertVariant.warning}
                        isInline
                        actionClose={actionClose}
                    >
                        {error.message}
                    </Alert>
                );
            });
    }, []);

    const isFetching = isFetchingPrimary || isFetchingSecondary;

    function onClickCreate() {
        history.push(getEntityPath(entityType, undefined, { action: 'create' }));
    }

    function handleDelete(idDelete: string) {
        return deletePermissionSet(idDelete).then(() => {
            // Remove the deleted entity.
            setPermissionSets(permissionSets.filter(({ id }) => id !== idDelete));
        }); // TODO catch error display alert
    }

    function handleEdit() {
        history.push(getEntityPath(entityType, entityId, { action: 'update' }));
    }

    function handleCancel() {
        // Go back from action=create to list or go back from action=update to entity.
        history.goBack();
    }

    function handleSubmit(values: PermissionSet): Promise<null> {
        return action === 'create'
            ? createPermissionSet(values).then((entityCreated) => {
                  // Append the created entity.
                  setPermissionSets([...permissionSets, entityCreated]);

                  // Replace path which had action=create with plain entity path.
                  history.replace(getEntityPath(entityType, entityCreated.id));

                  return null; // because the form has only catch and finally
              })
            : updatePermissionSet(values).then(() => {
                  // Replace the updated entity.
                  setPermissionSets(
                      permissionSets.map((entity) => (entity.id === values.id ? values : entity))
                  );

                  // Replace path which had action=update with plain entity path.
                  history.replace(getEntityPath(entityType, entityId));

                  return null; // because the form has only catch and finally
              });
    }

    const permissionSet =
        permissionSets.find(({ id }) => id === entityId) || getNewPermissionSet(resources);
    const isActionable = !defaultRoles[permissionSet.name];
    const hasAction = Boolean(action);
    const isEntity = hasAction || Boolean(entityId);

    return (
        <>
            <AccessControlPageTitle entityType={entityType} isEntity={isEntity} />
            <AccessControlNav entityType={entityType} />
            {alertPermissionSets}
            {alertResources}
            {alertRoles}
            {isFetching ? (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            ) : isEntity ? (
                <PermissionSetForm
                    isActionable={isActionable}
                    action={action}
                    permissionSet={getCompletePermissionSet(permissionSet, resources)}
                    permissionSets={permissionSets}
                    handleCancel={handleCancel}
                    handleEdit={handleEdit}
                    handleSubmit={handleSubmit}
                />
            ) : (
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
                                <Button
                                    variant="primary"
                                    onClick={onClickCreate}
                                    isDisabled={isFetching || resources.length === 0}
                                    isSmall
                                >
                                    Create permission set
                                </Button>
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                    {permissionSets.length !== 0 && (
                        <PermissionSetsList
                            entityId={entityId}
                            permissionSets={permissionSets}
                            roles={roles}
                            handleDelete={handleDelete}
                        />
                    )}
                </>
            )}
        </>
    );
}

export default PermissionSets;
