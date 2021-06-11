/* eslint-disable react/jsx-no-bind */
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

import { defaultRoles } from 'constants/accessControl';
import {
    AccessScope,
    PermissionSet,
    Role,
    createRole,
    // deleteRole,
    fetchAccessScopes,
    fetchPermissionSets,
    fetchRolesAsArray,
    updateRole,
} from 'services/RolesService';

import AccessControlNav from '../AccessControlNav';
import { getEntityPath, getQueryObject } from '../accessControlPaths';

import RoleForm from './RoleForm';
import RolesList from './RolesList';

const entityType = 'ROLE';

const roleNew: Role = {
    name: '',
    resourceToAccess: {},
    id: '',
    description: '',
    permissionSetId: '',
    accessScopeId: '',
};

function Roles(): ReactElement {
    const history = useHistory();
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;
    const { entityId: entityName } = useParams(); // identify role by name in routes

    const [isFetching, setIsFetching] = useState(false);
    const [roles, setRoles] = useState<Role[]>([]);
    const [alertRoles, setAlertRoles] = useState<ReactElement | null>(null);
    const [permissionSets, setPermissionSets] = useState<PermissionSet[]>([]);
    const [alertPermissionSets, setAlertPermissionSets] = useState<ReactElement | null>(null);
    const [accessScopes, setAccessScopes] = useState<AccessScope[]>([]);
    const [alertAccessScopes, setAlertAccessScopes] = useState<ReactElement | null>(null);

    useEffect(() => {
        // The primary request has fetching spinner and unclosable alert.
        setIsFetching(true);
        setAlertRoles(null);
        fetchRolesAsArray()
            .then((rolesFetched) => {
                setRoles(rolesFetched);
            })
            .catch((error) => {
                setAlertRoles(
                    <Alert title="Fetch roles failed" variant={AlertVariant.danger} isInline>
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setIsFetching(false);
            });

        // TODO Until secondary requests succeed, disable Create and Edit because selections might be incomplete?
        setAlertPermissionSets(null);
        fetchPermissionSets()
            .then((permissionSetsFetched) => {
                setPermissionSets(permissionSetsFetched);
            })
            .catch((error) => {
                const actionClose = <AlertActionCloseButton onClose={() => setAlertRoles(null)} />;
                setAlertPermissionSets(
                    <Alert
                        title="Fetch permission sets failed"
                        variant={AlertVariant.warning}
                        isInline
                        actionClose={actionClose}
                    >
                        {error.message}
                    </Alert>
                );
            });

        setAlertAccessScopes(null);
        fetchAccessScopes()
            .then((accessScopesFetched) => {
                setAccessScopes(accessScopesFetched);
            })
            .catch((error) => {
                const actionClose = <AlertActionCloseButton onClose={() => setAlertRoles(null)} />;
                setAlertAccessScopes(
                    <Alert
                        title="Fetch access scopes failed"
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
        history.push(getEntityPath(entityType, entityName, { ...queryObject, action: 'update' }));
    }

    function onClickCancel() {
        // The entityName is undefined for create and defined for update.
        history.push(getEntityPath(entityType, entityName, { ...queryObject, action: undefined }));
    }

    function submitValues(values: Role): Promise<null> {
        return action === 'create'
            ? createRole(values).then(() => {
                  // Append the values, because backend does not assign an id to the role.
                  setRoles([...roles, values]);

                  // Clear the action and also any filtering (in case the created entity does not match).
                  history.push(getEntityPath(entityType, values.name));

                  return null; // because the form has only catch and finally
              })
            : updateRole(values).then(() => {
                  // Replace the updated entity.
                  setRoles(roles.map((entity) => (entity.name === values.name ? values : entity)));

                  // Clear the action and also any filtering (in case the updated entity does not match).
                  history.push(getEntityPath(entityType, entityName));

                  return null; // because the form has only catch and finally
              });
    }

    const role = roles.find(({ name }) => name === entityName) || roleNew;
    const isActionable = !defaultRoles[role.name];
    const hasAction = Boolean(action);
    const isExpanded = hasAction || Boolean(entityName);

    const panelContent = (
        <DrawerPanelContent minSize="90%">
            <DrawerHead>
                <Title headingLevel="h3">{action === 'create' ? 'Create role' : role.name}</Title>
                {!hasAction && (
                    <DrawerActions>
                        <DrawerCloseButton onClick={onClickClose} />
                    </DrawerActions>
                )}
            </DrawerHead>
            <DrawerPanelBody>
                <RoleForm
                    isActionable={isActionable}
                    action={action}
                    role={role}
                    permissionSets={permissionSets}
                    accessScopes={accessScopes}
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
            {alertRoles}
            {alertPermissionSets}
            {alertAccessScopes}
            {isFetching ? (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            ) : (
                <Drawer isExpanded={isExpanded} style={{ height: 'auto' }}>
                    <DrawerContent panelContent={panelContent}>
                        <DrawerContentBody>
                            <Toolbar inset={{ default: 'insetNone' }}>
                                <ToolbarContent>
                                    <ToolbarItem>
                                        <Button
                                            variant="primary"
                                            onClick={onClickCreate}
                                            isDisabled={isExpanded || isFetching}
                                            isSmall
                                        >
                                            Create role
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarContent>
                            </Toolbar>
                            <RolesList
                                entityName={entityName}
                                roles={roles}
                                permissionSets={permissionSets}
                                accessScopes={accessScopes}
                            />
                        </DrawerContentBody>
                    </DrawerContent>
                </Drawer>
            )}
        </>
    );
}

export default Roles;
