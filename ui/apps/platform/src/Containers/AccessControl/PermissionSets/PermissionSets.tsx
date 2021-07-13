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

import { getIsDefaultRoleName } from 'constants/accessControl';
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

import AccessControlDescription from '../AccessControlDescription';
import AccessControlHeading from '../AccessControlHeading';
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

    const [counterFetching, setCounterFetching] = useState(0);

    const [permissionSets, setPermissionSets] = useState<PermissionSet[]>([]);
    const [alertPermissionSets, setAlertPermissionSets] = useState<ReactElement | null>(null);

    const [resources, setResources] = useState<string[]>([]);
    const [alertResources, setAlertResources] = useState<ReactElement | null>(null);

    const [roles, setRoles] = useState<Role[]>([]);
    const [alertRoles, setAlertRoles] = useState<ReactElement | null>(null);

    useEffect(() => {
        // The primary request has an unclosable alert.
        setCounterFetching((counterPrev) => counterPrev + 1);
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
                setCounterFetching((counterPrev) => counterPrev - 1);
            });

        // The secondary requests have closable alerts.

        setCounterFetching((counterPrev) => counterPrev + 1);
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
                setCounterFetching((counterPrev) => counterPrev - 1);
            });

        setCounterFetching((counterPrev) => counterPrev + 1);
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
            })
            .finally(() => {
                setCounterFetching((counterPrev) => counterPrev - 1);
            });
    }, []);

    function handleCreate() {
        history.push(getEntityPath(entityType, undefined, { action: 'create' }));
    }

    function handleDelete(idDelete: string) {
        return deletePermissionSet(idDelete).then(() => {
            // Remove the deleted entity.
            setPermissionSets(permissionSets.filter(({ id }) => id !== idDelete));
        }); // list has catch
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
    const isActionable = !getIsDefaultRoleName(permissionSet.name);
    const hasAction = Boolean(action);
    const isEntity = hasAction || Boolean(entityId);

    return (
        <>
            <AccessControlPageTitle entityType={entityType} isEntity={isEntity} />
            <AccessControlHeading
                entityType={entityType}
                entityName={
                    permissionSet &&
                    (action === 'create' ? 'Add permission set' : permissionSet.name)
                }
                isDisabled={hasAction}
            />
            <AccessControlNav entityType={entityType} isDisabled={hasAction} />
            <AccessControlDescription>
                Add predefined sets of application level permissions that users have when
                interacting with the platform
            </AccessControlDescription>
            {alertPermissionSets}
            {alertResources}
            {alertRoles}
            {counterFetching !== 0 ? (
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
                <PermissionSetsList
                    permissionSets={permissionSets}
                    roles={roles}
                    handleCreate={handleCreate}
                    handleDelete={handleDelete}
                />
            )}
        </>
    );
}

export default PermissionSets;
