/* eslint-disable react/jsx-no-bind */
import React, { ReactElement, useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import {
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
import { actions as authActions, types as authActionTypes } from 'reducers/auth';
import { actions as roleActions, types as roleActionTypes } from 'reducers/roles';
import { AuthProvider, createAuthProvider, updateAuthProvider } from 'services/AuthService';

import { getEntityPath, getQueryObject } from '../accessControlPaths';

import AccessControlNav from '../AccessControlNav';
import AccessControlPageTitle from '../AccessControlPageTitle';

import AuthProviderForm from './AuthProviderForm';
import AuthProvidersList from './AuthProvidersList';

const entityType = 'AUTH_PROVIDER';

const authProviderNew = {
    id: '',
    name: '',
    type: 'oidc',
    config: {},
} as AuthProvider; // TODO what are the minimum properties for create request?

const authProviderState = createStructuredSelector({
    authProviders: selectors.getAvailableAuthProviders,
    roles: selectors.getRoles,
    isFetchingAuthProviders: (state) =>
        selectors.getLoadingStatus(state, authActionTypes.FETCH_AUTH_PROVIDERS) as boolean,
    isFetchingRoles: (state) =>
        selectors.getLoadingStatus(state, roleActionTypes.FETCH_ROLES) as boolean,
});

function getNewAuthProviderObj(type) {
    return { ...authProviderNew, type };
}

function AuthProviders(): ReactElement {
    const history = useHistory();
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action, type } = queryObject;
    const { entityId } = useParams();
    const dispatch = useDispatch();

    const [isCreateMenuOpen, setIsCreateMenuOpen] = useState(false);
    const { authProviders, roles, isFetchingAuthProviders, isFetchingRoles } = useSelector(
        authProviderState
    );

    useEffect(() => {
        dispatch(authActions.fetchAuthProviders.request());
        dispatch(roleActions.fetchRoles.request());
    }, [dispatch]);

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
                  //   setAuthProviders([...authProviders, entityCreated]);

                  // Replace path which had action=create with plain entity path.
                  history.replace(getEntityPath(entityType, entityCreated.id));

                  return entityCreated;
              })
            : updateAuthProvider(values).then((entityUpdated) => {
                  // Replace the updated entity.
                  //   setAuthProviders(
                  //       authProviders.map((entity) =>
                  //           entity.id === entityUpdated.id ? entityUpdated : entity
                  //       )
                  //   );

                  // Replace path which had action=update with plain entity path.
                  history.replace(getEntityPath(entityType, entityId));

                  return entityUpdated;
              });
    }

    const selectedAuthProvider =
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
            {(isFetchingAuthProviders || isFetchingRoles) && (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            )}
            {!isFetchingAuthProviders && !isFetchingRoles && isExpanded && (
                <AuthProviderForm
                    isActionable={isActionable}
                    action={action}
                    selectedAuthProvider={selectedAuthProvider}
                    roles={roles}
                    onClickCancel={onClickCancel}
                    onClickEdit={onClickEdit}
                    submitValues={submitValues}
                />
            )}
            {!isFetchingAuthProviders && !isFetchingRoles && !isExpanded && (
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
                                    className="pf-m-small"
                                    onSelect={onClickCreate}
                                    position={DropdownPosition.right}
                                    toggle={
                                        <DropdownToggle
                                            onToggle={onToggleCreateMenu}
                                            toggleIndicator={CaretDownIcon}
                                            isPrimary
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
