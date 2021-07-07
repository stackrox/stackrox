/* eslint-disable no-nested-ternary */
/* eslint-disable react/jsx-no-bind */
import React, { ReactElement, useEffect, useState } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertVariant,
    Bullseye,
    Spinner,
} from '@patternfly/react-core';

import { defaultRoleDescriptions, getIsDefaultRoleName } from 'constants/accessControl';
import { Group } from 'services/AuthService';
import { fetchGroups } from 'services/GroupsService';
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

import AccessControlHeading from '../AccessControlHeading';
import AccessControlNav from '../AccessControlNav';
import AccessControlPageTitle from '../AccessControlPageTitle';
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

    const [counterFetching, setCounterFetching] = useState(0);

    const [roles, setRoles] = useState<Role[]>([]);
    const [alertRoles, setAlertRoles] = useState<ReactElement | null>(null);

    const [groups, setGroups] = useState<Group[]>([]);
    const [alertGroups, setAlertGroups] = useState<ReactElement | null>(null);

    const [permissionSets, setPermissionSets] = useState<PermissionSet[]>([]);
    const [alertPermissionSets, setAlertPermissionSets] = useState<ReactElement | null>(null);

    const [accessScopes, setAccessScopes] = useState<AccessScope[]>([]);
    const [alertAccessScopes, setAlertAccessScopes] = useState<ReactElement | null>(null);

    useEffect(() => {
        // The primary request has unclosable alert.
        setCounterFetching((counterPrev) => counterPrev + 1);
        setAlertRoles(null);
        fetchRolesAsArray()
            .then((rolesFetched) => {
                // Provide descriptions for default roles until backend returns them.
                setRoles(
                    rolesFetched.map((role) =>
                        getIsDefaultRoleName(role.name)
                            ? { ...role, description: defaultRoleDescriptions[role.name] }
                            : role
                    )
                );
            })
            .catch((error) => {
                setAlertRoles(
                    <Alert title="Fetch roles failed" variant={AlertVariant.danger} isInline>
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setCounterFetching((counterPrev) => counterPrev - 1);
            });

        // The secondary requests have closable alerts.

        setCounterFetching((counterPrev) => counterPrev + 1);
        setAlertGroups(null);
        fetchGroups()
            .then((dataFetched) => {
                console.log(dataFetched.response.groups); // eslint-disable-line
                setGroups(dataFetched.response.groups);
            })
            .catch((error) => {
                const actionClose = <AlertActionCloseButton onClose={() => setAlertGroups(null)} />;
                setAlertGroups(
                    <Alert
                        title="Fetch groups failed"
                        variant={AlertVariant.warning}
                        isInline
                        actionClose={actionClose}
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setCounterFetching((counterPrev) => counterPrev - 1);
            });

        setCounterFetching((counterPrev) => counterPrev + 1);
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
            })
            .finally(() => {
                setCounterFetching((counterPrev) => counterPrev - 1);
            });

        setCounterFetching((counterPrev) => counterPrev + 1);
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
            })
            .finally(() => {
                setCounterFetching((counterPrev) => counterPrev - 1);
            });
    }, []);

    function handleCreate() {
        history.push(getEntityPath(entityType, undefined, { action: 'create' }));
    }

    function handleDelete(nameDelete: string) {
        return deleteRole(nameDelete).then(() => {
            // Remove the deleted entity.
            setRoles(roles.filter(({ name }) => name !== nameDelete));
        }); // list has catch
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
    const isActionable = !getIsDefaultRoleName(role.name);
    const hasAction = Boolean(action);
    const isEntity = hasAction || Boolean(entityName);

    return (
        <>
            <AccessControlPageTitle entityType={entityType} isEntity={isEntity} />
            <AccessControlHeading
                entityType={entityType}
                entityName={role && (action === 'create' ? 'Add role' : role.name)}
                isDisabled={hasAction}
            />
            <AccessControlNav entityType={entityType} isDisabled={hasAction} />
            {alertRoles}
            {alertPermissionSets}
            {alertAccessScopes}
            {alertGroups}
            {counterFetching !== 0 ? (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            ) : isEntity ? (
                <RoleForm
                    isActionable={isActionable}
                    action={action}
                    role={role}
                    roles={roles}
                    permissionSets={permissionSets}
                    accessScopes={accessScopes}
                    handleCancel={handleCancel}
                    handleEdit={handleEdit}
                    handleSubmit={handleSubmit}
                />
            ) : (
                <RolesList
                    roles={roles}
                    groups={groups}
                    permissionSets={permissionSets}
                    accessScopes={accessScopes}
                    handleCreate={handleCreate}
                    handleDelete={handleDelete}
                />
            )}
        </>
    );
}

export default Roles;
