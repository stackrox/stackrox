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
    PermissionSet,
    Role,
    createPermissionSet,
    deletePermissionSet,
    fetchPermissionSets,
    fetchResources,
    fetchRolesAsArray,
    updatePermissionSet,
} from 'services/RolesService';

import AccessControlDescription from '../AccessControlDescription';
import AccessControlPageTitle from '../AccessControlPageTitle';
import { getEntityPath, getQueryObject } from '../accessControlPaths';

import PermissionSetForm from './PermissionSetForm';
import PermissionSetsList from './PermissionSetsList';
import { getNewPermissionSet, getCompletePermissionSet } from './permissionSets.utils';
import AccessControlHeaderActionBar from '../AccessControlHeaderActionBar';
import AccessControlBreadcrumbs from '../AccessControlBreadcrumbs';
import AccessControlHeading from '../AccessControlHeading';
import AccessControlNoPermission from '../AccessControlNoPermission';
import usePermissions from '../../../hooks/usePermissions';
import { isUserResource } from '../traits';

const entityType = 'PERMISSION_SET';

function PermissionSets(): ReactElement {
    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasReadAccessForPage = hasReadAccess('Access');
    const hasWriteAccessForPage = hasReadWriteAccess('Access');
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
        fetchResources()
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

    // Return "no access" page immediately if user doesn't have enough permissions.
    if (!hasReadAccessForPage) {
        return (
            <>
                <AccessControlNoPermission subPage="permission sets" entityType={entityType} />
            </>
        );
    }

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
        history.push(getEntityPath(entityType, entityId, { action: 'edit' }));
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

                  // Go back from action=create to list.
                  history.goBack();

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

    const permissionSet = permissionSets.find(({ id }) => id === entityId);
    const hasAction = Boolean(action);
    const isList = typeof entityId !== 'string' && !hasAction;

    return (
        <>
            <AccessControlPageTitle entityType={entityType} isList={isList} />
            {isList ? (
                <>
                    <AccessControlHeading entityType={entityType} />
                    <AccessControlHeaderActionBar
                        displayComponent={
                            <AccessControlDescription>
                                Create predefined sets of application level permissions that users
                                have when interacting with the platform
                            </AccessControlDescription>
                        }
                        actionComponent={
                            <Button
                                isDisabled={!hasWriteAccessForPage}
                                variant="primary"
                                onClick={handleCreate}
                            >
                                Create permission set
                            </Button>
                        }
                    />
                </>
            ) : (
                <AccessControlBreadcrumbs
                    entityType={entityType}
                    entityName={action === 'create' ? 'Create permission set' : permissionSet?.name}
                />
            )}
            {alertPermissionSets}
            {alertResources}
            {alertRoles}
            <PageSection variant={isList ? PageSectionVariants.default : PageSectionVariants.light}>
                {counterFetching !== 0 ? (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                ) : isList ? (
                    <PermissionSetsList
                        permissionSets={permissionSets}
                        roles={roles}
                        handleCreate={handleCreate}
                        handleDelete={handleDelete}
                    />
                ) : typeof entityId === 'string' && !permissionSet ? (
                    <NotFoundMessage
                        title="Permission set does not exist"
                        message={`Permission set id: ${entityId}`}
                        actionText="Permission sets"
                        url={getEntityPath(entityType)}
                    />
                ) : (
                    <PermissionSetForm
                        isActionable={!permissionSet || isUserResource(permissionSet.traits)}
                        action={action}
                        permissionSet={
                            permissionSet
                                ? getCompletePermissionSet(permissionSet, resources)
                                : getNewPermissionSet(resources)
                        }
                        permissionSets={permissionSets}
                        handleCancel={handleCancel}
                        handleEdit={handleEdit}
                        handleSubmit={handleSubmit}
                    />
                )}
            </PageSection>
        </>
    );
}

export default PermissionSets;
