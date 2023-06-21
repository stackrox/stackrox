/* eslint-disable no-nested-ternary */
import React, { ReactElement, useEffect, useState } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertVariant,
    Bullseye,
    Button,
    PageSection,
    PageSectionVariants,
    Spinner,
} from '@patternfly/react-core';

import NotFoundMessage from 'Components/NotFoundMessage';
import {
    AccessScope,
    fetchAccessScopes,
    defaultAccessScopeIds,
} from 'services/AccessScopesService';
import { Group, fetchAuthProviders } from 'services/AuthService';
import { fetchGroups } from 'services/GroupsService';
import {
    PermissionSet,
    Role,
    createRole,
    deleteRole,
    fetchPermissionSets,
    fetchRolesAsArray,
    updateRole,
} from 'services/RolesService';

import AccessControlDescription from '../AccessControlDescription';
import AccessControlPageTitle from '../AccessControlPageTitle';
import { getEntityPath, getQueryObject } from '../accessControlPaths';

import RoleForm from './RoleForm';
import RolesList from './RolesList';
import AccessControlBreadcrumbs from '../AccessControlBreadcrumbs';
import AccessControlHeaderActionBar from '../AccessControlHeaderActionBar';
import AccessControlHeading from '../AccessControlHeading';
import usePermissions from '../../../hooks/usePermissions';
import AccessControlNoPermission from '../AccessControlNoPermission';
import { isUserResource } from '../traits';

const entityType = 'ROLE';

function Roles(): ReactElement {
    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasReadAccessForPage = hasReadAccess('Access');
    const hasWriteAccessForPage = hasReadWriteAccess('Access');
    const history = useHistory();
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action, s } = queryObject;
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

    function getDefaultAccessScopeID() {
        return defaultAccessScopeIds.Unrestricted;
    }

    const roleNew: Role = {
        name: '',
        resourceToAccess: {},
        description: '',
        permissionSetId: '',
        accessScopeId: getDefaultAccessScopeID(),
    };

    useEffect(() => {
        // The primary request has unclosable alert.
        setCounterFetching((counterPrev) => counterPrev + 1);
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
                setCounterFetching((counterPrev) => counterPrev - 1);
            });

        // The secondary requests have closable alerts.

        setCounterFetching((counterPrev) => counterPrev + 1);
        setAlertGroups(null);
        Promise.all([fetchGroups(), fetchAuthProviders()])
            .then(([dataFetchedGroups, dataFetchedAuthProviders]) => {
                const groupsFetched = dataFetchedGroups.response.groups;
                const authProvidersFetched = dataFetchedAuthProviders.response;

                // Filter out any groups which refer to obsolete auth providers,
                // so role is deletable if not referenced by any current auth provider
                // as either its minimum access role or as a role in assigned rules.
                const groupsFiltered = groupsFetched.filter((group) => {
                    if (!group.props) {
                        return true;
                    }

                    const { authProviderId } = group.props;
                    return authProvidersFetched.some(({ id }) => id === authProviderId);
                });

                setGroups(groupsFiltered);
            })
            .catch((error) => {
                const actionClose = <AlertActionCloseButton onClose={() => setAlertGroups(null)} />;
                setAlertGroups(
                    <Alert
                        title="Fetch auth providers or groups failed"
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

    // Return "no access" page immediately if user doesn't have enough permissions.
    if (!hasReadAccessForPage) {
        return (
            <>
                <AccessControlNoPermission subPage="roles" entityType={entityType} />
            </>
        );
    }

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
        history.push(getEntityPath(entityType, entityName, { action: 'edit' }));
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

                  // Go back from action=create to list.
                  history.goBack();

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

    const role = roles.find(({ name }) => name === entityName);
    const hasAction = Boolean(action);
    const isList = typeof entityName !== 'string' && !hasAction;

    return (
        <>
            <AccessControlPageTitle entityType={entityType} isList={isList} />
            {isList ? (
                <>
                    <AccessControlHeading entityType={entityType} />
                    <AccessControlHeaderActionBar
                        displayComponent={
                            <AccessControlDescription>
                                Create user roles by selecting the permission sets and access scopes
                                required for user&apos;s jobs
                            </AccessControlDescription>
                        }
                        actionComponent={
                            <Button
                                isDisabled={!hasWriteAccessForPage}
                                variant="primary"
                                onClick={handleCreate}
                            >
                                Create role
                            </Button>
                        }
                    />
                </>
            ) : (
                <AccessControlBreadcrumbs
                    entityType={entityType}
                    entityName={action === 'create' ? 'Create role' : role?.name}
                />
            )}
            <PageSection variant={isList ? PageSectionVariants.default : PageSectionVariants.light}>
                {alertRoles}
                {alertPermissionSets}
                {alertAccessScopes}
                {alertGroups}
                {counterFetching !== 0 ? (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                ) : isList ? (
                    <RolesList
                        roles={roles}
                        s={s}
                        groups={groups}
                        permissionSets={permissionSets}
                        accessScopes={accessScopes}
                        handleDelete={handleDelete}
                    />
                ) : typeof entityName === 'string' && !role ? (
                    <NotFoundMessage
                        title="Role does not exist"
                        message={`Role name: ${entityName}`}
                        actionText="Roles"
                        url={getEntityPath(entityType)}
                    />
                ) : (
                    <RoleForm
                        isActionable={!role || isUserResource(role.traits)}
                        action={action}
                        role={role ?? roleNew}
                        roles={roles}
                        permissionSets={permissionSets}
                        accessScopes={accessScopes}
                        handleCancel={handleCancel}
                        handleEdit={handleEdit}
                        handleSubmit={handleSubmit}
                    />
                )}
            </PageSection>
        </>
    );
}

export default Roles;
