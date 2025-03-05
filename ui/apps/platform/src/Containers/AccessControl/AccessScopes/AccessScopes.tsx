/* eslint-disable no-nested-ternary */
import React, { ReactElement, useEffect, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    Bullseye,
    Button,
    PageSection,
    Spinner,
} from '@patternfly/react-core';

import NotFoundMessage from 'Components/NotFoundMessage';
import {
    AccessScope,
    accessScopeNew,
    createAccessScope,
    deleteAccessScope,
    fetchAccessScopes,
    updateAccessScope,
} from 'services/AccessScopesService';
import { Role, fetchRolesAsArray } from 'services/RolesService';

import AccessControlDescription from '../AccessControlDescription';
import AccessControlPageTitle from '../AccessControlPageTitle';
import { getEntityPath, getQueryObject } from '../accessControlPaths';

import AccessScopeFormWrapper from './AccessScopeFormWrapper';
import AccessScopesList from './AccessScopesList';

import './AccessScopes.css';
import AccessControlHeading from '../AccessControlHeading';
import AccessControlBreadcrumbs from '../AccessControlBreadcrumbs';
import AccessControlHeaderActionBar from '../AccessControlHeaderActionBar';
import usePermissions from '../../../hooks/usePermissions';
import { isUserResource } from '../traits';

const entityType = 'ACCESS_SCOPE';

function AccessScopes(): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPage = hasReadWriteAccess('Access');
    const navigate = useNavigate();
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;
    const { entityId } = useParams();

    const [counterFetching, setCounterFetching] = useState(0);

    const [accessScopes, setAccessScopes] = useState<AccessScope[]>([]);
    const [alertAccessScopes, setAlertAccessScopes] = useState<ReactElement | null>(null);

    const [roles, setRoles] = useState<Role[]>([]);
    const [alertRoles, setAlertRoles] = useState<ReactElement | null>(null);

    useEffect(() => {
        // The primary request has an unclosable alert.
        setCounterFetching((counterPrev) => counterPrev + 1);
        setAlertAccessScopes(null);
        fetchAccessScopes()
            .then((accessScopesFetched) => {
                setAccessScopes(accessScopesFetched);
            })
            .catch((error) => {
                setAlertAccessScopes(
                    <Alert
                        title="Fetch access scopes failed"
                        component="p"
                        variant="danger"
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
                        component="p"
                        variant="warning"
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
        navigate(getEntityPath(entityType, undefined, { action: 'create' }));
    }

    function handleDelete(idDelete: string) {
        return deleteAccessScope(idDelete).then(() => {
            // Remove the deleted entity.
            setAccessScopes(accessScopes.filter(({ id }) => id !== idDelete));
        }); // list has catch
    }

    function handleEdit() {
        navigate(getEntityPath(entityType, entityId, { action: 'edit' }));
    }

    function handleCancel() {
        // Go back from action=create to list or go back from action=update to entity.
        navigate(-1);
    }

    function handleSubmit(values: AccessScope): Promise<null> {
        return action === 'create'
            ? createAccessScope(values).then((entityCreated) => {
                  // Append the created entity.
                  setAccessScopes([...accessScopes, entityCreated]);

                  // Go back from action=create to list.
                  navigate(-1);

                  return null; // because the form has only catch and finally
              })
            : updateAccessScope(values).then(() => {
                  // Replace the updated entity with values because response is empty object.
                  setAccessScopes(
                      accessScopes.map((entity) => (entity.id === values.id ? values : entity))
                  );

                  // Replace path which had action=update with plain entity path.
                  navigate(getEntityPath(entityType, entityId), { replace: true });

                  return null; // because the form has only catch and finally
              });
    }

    const accessScope = accessScopes.find(({ id }) => id === entityId);
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
                                Create predefined sets of authorized Kubernetes resources that users
                                should be able to access
                            </AccessControlDescription>
                        }
                        actionComponent={
                            <Button
                                isDisabled={!hasWriteAccessForPage}
                                variant="primary"
                                onClick={handleCreate}
                            >
                                Create access scope
                            </Button>
                        }
                    />
                </>
            ) : (
                <AccessControlBreadcrumbs
                    entityType={entityType}
                    entityName={action === 'create' ? 'Create access scope' : accessScope?.name}
                />
            )}
            {alertAccessScopes}
            {alertRoles}
            <PageSection variant={isList ? 'default' : 'light'}>
                {counterFetching !== 0 ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : isList ? (
                    <AccessScopesList
                        accessScopes={accessScopes}
                        roles={roles}
                        handleDelete={handleDelete}
                    />
                ) : typeof entityId === 'string' && !accessScope ? (
                    <NotFoundMessage
                        title="Access scope does not exist"
                        message={`Access scope id: ${entityId}`}
                        actionText="Access scopes"
                        url={getEntityPath(entityType)}
                    />
                ) : (
                    <AccessScopeFormWrapper
                        isActionable={!accessScope || isUserResource(accessScope.traits)}
                        action={action}
                        accessScope={accessScope ?? accessScopeNew}
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

export default AccessScopes;
