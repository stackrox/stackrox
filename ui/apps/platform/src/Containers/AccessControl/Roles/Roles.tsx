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
    AccessScope,
    PermissionSet,
    Role,
    createRole,
    deleteRole,
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

    function onClickCreate() {
        history.push(getEntityPath(entityType, undefined, { action: 'create' }));
    }

    function handleDelete(nameDelete: string) {
        return deleteRole(nameDelete).then(() => {
            // Remove the deleted entity.
            setRoles(roles.filter(({ name }) => name !== nameDelete));
        }); // TODO catch error display alert
    }

    function handleEdit() {
        history.push(getEntityPath(entityType, entityName, { action: 'update' }));
    }

    function handleCancel() {
        // Go back from action=create to list or go back from action=update to entity.
        history.goBack();
    }

    function handleSubmit(values: Role): Promise<null> {
        return action === 'create'
            ? createRole(values).then(() => {
                  // Append the values, because backend does not assign an id to the role.
                  setRoles([...roles, values]);

                  // Replace path which had action=create with plain entity path.
                  history.replace(getEntityPath(entityType, values.name));

                  return null; // because the form has only catch and finally
              })
            : updateRole(values).then(() => {
                  // Replace the updated entity.
                  setRoles(roles.map((entity) => (entity.name === values.name ? values : entity)));

                  // Replace path which had action=update with plain entity path.
                  history.replace(getEntityPath(entityType, entityName));

                  return null; // because the form has only catch and finally
              });
    }

    const role = roles.find(({ name }) => name === entityName) || roleNew;
    const isActionable = !defaultRoles[role.name];
    const hasAction = Boolean(action);
    const isExpanded = hasAction || Boolean(entityName);

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
            ) : isExpanded ? (
                <RoleForm
                    isActionable={isActionable}
                    action={action}
                    role={role}
                    permissionSets={permissionSets}
                    accessScopes={accessScopes}
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
                                    <Title headingLevel="h2">Roles</Title>
                                </ToolbarItem>
                                <ToolbarItem>
                                    <Badge isRead>{roles.length}</Badge>
                                </ToolbarItem>
                            </ToolbarGroup>
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
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
                    {roles.length !== 0 && (
                        <RolesList
                            entityName={entityName}
                            roles={roles}
                            permissionSets={permissionSets}
                            accessScopes={accessScopes}
                            handleDelete={handleDelete}
                        />
                    )}
                </>
            )}
        </>
    );
}

export default Roles;
