import React, { ReactElement, useEffect, useState } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertVariant,
    Bullseye,
    Button,
    Drawer,
    DrawerActions,
    DrawerCloseButton,
    DrawerContent,
    DrawerContentBody,
    DrawerHead,
    DrawerPanelBody,
    DrawerPanelContent,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import { AccessType as AccessLevel, fetchResources } from 'services/RolesService';

import AccessControlNav from '../AccessControlNav';
import { getEntityPath, getQueryObject } from '../accessControlPaths';
import {
    createPermissionSet,
    fetchPermissionSets,
    fetchRoles,
    PermissionSet,
    Role,
    updatePermissionSet,
} from '../accessControlTypes';

import PermissionSetForm from './PermissionSetForm';
import PermissionSetsList from './PermissionSetsList';

function getNewPermissionSet(resources: string[]): PermissionSet {
    const resourceIdToAccess: Record<string, AccessLevel> = {};
    resources.forEach((resource) => {
        resourceIdToAccess[resource] = 'NO_ACCESS';
    });

    return {
        id: '',
        name: '',
        description: '',
        minimumAccessLevel: 'NO_ACCESS',
        resourceIdToAccess,
    };
}

const entityType = 'PERMISSION_SET';

function PermissionSets(): ReactElement {
    const history = useHistory();
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;
    const { entityId } = useParams();

    const [isFetching, setIsFetching] = useState(false);
    const [permissionSets, setPermissionSets] = useState<PermissionSet[]>([]);
    const [alertPermissionSets, setAlertPermissionSets] = useState<ReactElement | null>(null);
    const [resources, setResources] = useState<string[]>([]);
    const [alertResources, setAlertResources] = useState<ReactElement | null>(null);
    const [roles, setRoles] = useState<Role[]>([]);
    const [alertRoles, setAlertRoles] = useState<ReactElement | null>(null);

    useEffect(() => {
        // The primary request has fetching spinner and unclosable alert.
        setIsFetching(true);
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
                setIsFetching(false);
            });

        // TODO Until secondary requests succeed, disable Create and Edit because selections might be incomplete?
        setAlertResources(null);
        fetchResources()
            .then((data) => {
                setResources(data.response.resources);
            })
            .catch((error) => {
                // eslint-disable-next-line react/jsx-no-bind
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
            });

        setAlertRoles(null);
        fetchRoles()
            .then((rolesFetched) => {
                setRoles(rolesFetched);
            })
            .catch((error) => {
                // eslint-disable-next-line react/jsx-no-bind
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

    function onClickClose() {
        history.push(getEntityPath(entityType, undefined, queryObject));
    }

    function onClickCreate() {
        history.push(getEntityPath(entityType, undefined, { ...queryObject, action: 'create' }));
    }

    function onClickEdit() {
        history.push(getEntityPath(entityType, entityId, { ...queryObject, action: 'update' }));
    }

    function onClickCancel() {
        // The entityId is undefined for create and defined for update.
        history.push(getEntityPath(entityType, entityId, { ...queryObject, action: undefined }));
    }

    function submitValues(values: PermissionSet): Promise<PermissionSet> {
        return action === 'create'
            ? createPermissionSet(values).then((entityCreated) => {
                  // Append the created entity.
                  setPermissionSets([...permissionSets, entityCreated]);

                  // Clear the action and also any filtering (in case the created entity does not match).
                  history.push(getEntityPath(entityType, entityCreated.id));

                  return entityCreated;
              })
            : updatePermissionSet(values).then((entityUpdated) => {
                  // Replace the updated entity.
                  setPermissionSets(
                      permissionSets.map((entity) =>
                          entity.id === entityUpdated.id ? entityUpdated : entity
                      )
                  );

                  // Clear the action and also any filtering (in case the updated entity does not match).
                  history.push(getEntityPath(entityType, entityId));

                  return entityUpdated;
              });
    }

    const permissionSet =
        permissionSets.find(({ id }) => id === entityId) || getNewPermissionSet(resources);
    const isActionable = true; // TODO does it depend on user role?
    const hasAction = Boolean(action);
    const isExpanded = hasAction || Boolean(entityId);

    const panelContent = (
        <DrawerPanelContent minSize="90%">
            <DrawerHead>
                <Title headingLevel="h3">
                    {action === 'create' ? 'Create permission set' : permissionSet.name}
                </Title>
                {!hasAction && (
                    <DrawerActions>
                        <DrawerCloseButton onClick={onClickClose} />
                    </DrawerActions>
                )}
            </DrawerHead>
            <DrawerPanelBody>
                <PermissionSetForm
                    isActionable={isActionable}
                    action={action}
                    permissionSet={permissionSet}
                    onClickCancel={onClickCancel}
                    onClickEdit={onClickEdit}
                    submitValues={submitValues}
                />
            </DrawerPanelBody>
        </DrawerPanelContent>
    );

    // TODO Display backdrop which covers nav links and drawer body during action.
    return (
        <>
            <AccessControlNav entityType={entityType} />
            {alertPermissionSets}
            {alertResources}
            {alertRoles}
            {isFetching ? (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            ) : (
                <Drawer isExpanded={isExpanded}>
                    <DrawerContent panelContent={panelContent}>
                        <DrawerContentBody>
                            <Toolbar inset={{ default: 'insetNone' }}>
                                <ToolbarContent>
                                    <ToolbarItem>
                                        <Button
                                            variant="primary"
                                            onClick={onClickCreate}
                                            isDisabled={
                                                isExpanded || isFetching || resources.length === 0
                                            }
                                            isSmall
                                        >
                                            Create permission set
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarContent>
                            </Toolbar>
                            <PermissionSetsList
                                entityId={entityId}
                                permissionSets={permissionSets}
                                roles={roles}
                            />
                        </DrawerContentBody>
                    </DrawerContent>
                </Drawer>
            )}
        </>
    );
}

export default PermissionSets;
