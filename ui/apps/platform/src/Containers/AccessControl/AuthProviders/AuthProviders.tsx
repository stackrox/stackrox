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

import { filterAuthProviders } from 'reducers/auth';
import {
    AuthProvider,
    createAuthProvider,
    fetchAuthProviders,
    updateAuthProvider,
} from 'services/AuthService';
import { Role, fetchRolesAsArray } from 'services/RolesService';

import { getEntityPath, getQueryObject } from '../accessControlPaths';

import AccessControlNav from '../AccessControlNav';

import AuthProviderForm from './AuthProviderForm';
import AuthProvidersList from './AuthProvidersList';

const entityType = 'AUTH_PROVIDER';

const authProviderNew = {
    id: '',
    name: '',
    type: 'auth0',
    config: {},
} as AuthProvider; // TODO what are the minimum properties for create request?

function AuthProviders(): ReactElement {
    const history = useHistory();
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;
    const { entityId } = useParams();

    const [isFetching, setIsFetching] = useState(false);
    const [authProviders, setAuthProviders] = useState<AuthProvider[]>([]);
    const [alertAuthProviders, setAlertAuthProviders] = useState<ReactElement | null>(null);
    const [roles, setRoles] = useState<Role[]>([]);
    const [alertRoles, setAlertRoles] = useState<ReactElement | null>(null);

    useEffect(() => {
        // The primary request has fetching spinner and unclosable alert.
        setIsFetching(true);
        setAlertAuthProviders(null);
        fetchAuthProviders()
            .then((data) => {
                setAuthProviders(filterAuthProviders(data.response)); // filter out Login with username/password
            })
            .catch((error) => {
                setAlertAuthProviders(
                    <Alert
                        title="Fetch auth providers failed"
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

    function submitValues(values: AuthProvider): Promise<AuthProvider> {
        // TODO research special case for update active auth provider
        // See saveAuthProvider function in AuthService.ts
        return action === 'create'
            ? createAuthProvider(values).then((entityCreated) => {
                  // Append the created entity.
                  setAuthProviders([...authProviders, entityCreated]);

                  // Clear the action and also any filtering (in case the created entity does not match).
                  history.push(getEntityPath(entityType, entityCreated.id));

                  return entityCreated;
              })
            : updateAuthProvider(values).then((entityUpdated) => {
                  // Replace the updated entity.
                  setAuthProviders(
                      authProviders.map((entity) =>
                          entity.id === entityUpdated.id ? entityUpdated : entity
                      )
                  );

                  // Clear the action and also any filtering (in case the updated entity does not match).
                  history.push(getEntityPath(entityType, entityId));

                  return entityUpdated;
              });
    }

    const authProvider = authProviders.find(({ id }) => id === entityId) || authProviderNew;
    const isActionable = true; // TODO does it depend on user role?
    const hasAction = Boolean(action);
    const isExpanded = hasAction || Boolean(entityId);

    const panelContent = (
        <DrawerPanelContent minSize="90%">
            <DrawerHead>
                <Title headingLevel="h3">
                    {action === 'create' ? 'Create auth provider' : authProvider.name}
                </Title>
                {!hasAction && (
                    <DrawerActions>
                        <DrawerCloseButton onClick={onClickClose} />
                    </DrawerActions>
                )}
            </DrawerHead>
            <DrawerPanelBody>
                <AuthProviderForm
                    isActionable={isActionable}
                    action={action}
                    authProvider={authProvider}
                    roles={roles}
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
            {alertAuthProviders}
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
                                            isDisabled={isExpanded || isFetching}
                                            isSmall
                                        >
                                            Create auth provider
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarContent>
                            </Toolbar>
                            <AuthProvidersList entityId={entityId} authProviders={authProviders} />
                        </DrawerContentBody>
                    </DrawerContent>
                </Drawer>
            )}
        </>
    );
}

export default AuthProviders;
