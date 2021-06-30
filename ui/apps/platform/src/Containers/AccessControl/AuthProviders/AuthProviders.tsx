/* eslint-disable react/jsx-no-bind */
import React, { ReactElement, useEffect, useState } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    AlertVariant,
    Badge,
    Bullseye,
    Dropdown,
    DropdownItem,
    DropdownPosition,
    DropdownToggle,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

import { availableAuthProviders } from 'constants/accessControl';
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
import AccessControlPageTitle from '../AccessControlPageTitle';

import AuthProviderForm from './AuthProviderForm';
import AuthProvidersList from './AuthProvidersList';

const entityType = 'AUTH_PROVIDER';

const authProviderNew = {
    id: '',
    name: '',
    type: '',
    config: {},
} as AuthProvider; // TODO what are the minimum properties for create request?

function getNewAuthProviderObj(type) {
    return { ...authProviderNew, type };
}

function AuthProviders(): ReactElement {
    const history = useHistory();
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action, type } = queryObject;
    const { entityId } = useParams();

    const [isFetching, setIsFetching] = useState(false);
    const [isCreateMenuOpen, setIsCreateMenuOpen] = useState(false);
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

    function onToggleCreateMenu(isOpen) {
        setIsCreateMenuOpen(isOpen);
    }

    function onClickCreate(event) {
        history.push(
            getEntityPath(entityType, undefined, {
                ...queryObject,
                action: 'create',
                type: event?.target?.value,
            })
        );
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

                  // Replace path which had action=create with plain entity path.
                  history.replace(getEntityPath(entityType, entityCreated.id));

                  return entityCreated;
              })
            : updateAuthProvider(values).then((entityUpdated) => {
                  // Replace the updated entity.
                  setAuthProviders(
                      authProviders.map((entity) =>
                          entity.id === entityUpdated.id ? entityUpdated : entity
                      )
                  );

                  // Replace path which had action=update with plain entity path.
                  history.replace(getEntityPath(entityType, entityId));

                  return entityUpdated;
              });
    }

    const authProvider =
        authProviders.find(({ id }) => id === entityId) || getNewAuthProviderObj(type);
    const isActionable = true; // TODO does it depend on user role?
    const hasAction = Boolean(action);
    const isExpanded = hasAction || Boolean(entityId);

    const dropdownItems = availableAuthProviders.map(({ value, label }) => (
        <DropdownItem key={value} value={value} component="button">
            {label}
        </DropdownItem>
    ));

    // TODO Display backdrop which covers nav links and drawer body during action.
    return (
        <>
            <AccessControlPageTitle entityType={entityType} isEntity={isExpanded} />
            <AccessControlNav entityType={entityType} />
            {alertAuthProviders}
            {alertRoles}
            {isFetching && (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            )}
            {!isFetching && isExpanded && (
                <AuthProviderForm
                    isActionable={isActionable}
                    action={action}
                    authProvider={authProvider}
                    roles={roles}
                    onClickCancel={onClickCancel}
                    onClickEdit={onClickEdit}
                    submitValues={submitValues}
                />
            )}
            {!isFetching && !isExpanded && (
                <>
                    <Toolbar inset={{ default: 'insetNone' }}>
                        <ToolbarContent>
                            <ToolbarItem>
                                <Title headingLevel="h2">Auth Providers</Title>
                            </ToolbarItem>
                            <ToolbarItem>
                                <Badge isRead>{authProviders.length}</Badge>
                            </ToolbarItem>
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <Dropdown
                                    onSelect={onClickCreate}
                                    position={DropdownPosition.right}
                                    toggle={
                                        <DropdownToggle
                                            onToggle={onToggleCreateMenu}
                                            toggleIndicator={CaretDownIcon}
                                            isPrimary
                                            isDisabled={isFetching}
                                        >
                                            Add auth provider
                                        </DropdownToggle>
                                    }
                                    isOpen={isCreateMenuOpen}
                                    dropdownItems={dropdownItems}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                    <AuthProvidersList entityId={entityId} authProviders={authProviders} />
                </>
            )}
        </>
    );
}

export default AuthProviders;
