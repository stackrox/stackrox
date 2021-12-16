/* eslint-disable no-nested-ternary */
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

import ACSEmptyState from 'Components/ACSEmptyState';
import NotFoundMessage from 'Components/NotFoundMessage';
import { actions as authActions, types as authActionTypes } from 'reducers/auth';
import { actions as groupActions } from 'reducers/groups';
import {
    actions as roleActions,
    types as roleActionTypes,
    getHasReadWritePermission,
} from 'reducers/roles';
import { AuthProvider } from 'services/AuthService';

import { getEntityPath, getQueryObject } from '../accessControlPaths';
import { mergeGroupsWithAuthProviders } from './authProviders.utils';

import AccessControlNav from '../AccessControlNav';
import AccessControlPageTitle from '../AccessControlPageTitle';

import AccessControlDescription from '../AccessControlDescription';
import AccessControlHeading from '../AccessControlHeading';
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
    groups: selectors.getRuleGroups,
    isFetchingAuthProviders: (state) =>
        selectors.getLoadingStatus(state, authActionTypes.FETCH_AUTH_PROVIDERS) as boolean,
    isFetchingRoles: (state) =>
        selectors.getLoadingStatus(state, roleActionTypes.FETCH_ROLES) as boolean,
    userRolePermissions: selectors.getUserRolePermissions,
    availableProviderTypes: selectors.getAvailableProviderTypes,
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
    const {
        authProviders,
        groups,
        isFetchingAuthProviders,
        isFetchingRoles,
        userRolePermissions,
        availableProviderTypes,
    } = useSelector(authProviderState);
    const hasWriteAccess = getHasReadWritePermission('AuthProvider', userRolePermissions);

    const authProvidersWithRules = mergeGroupsWithAuthProviders(authProviders, groups);

    useEffect(() => {
        dispatch(authActions.fetchAuthProviders.request());
        dispatch(roleActions.fetchRoles.request());
        dispatch(groupActions.fetchGroups.request());
    }, [dispatch]);

    function onToggleCreateMenu(isOpen) {
        setIsCreateMenuOpen(isOpen);
    }

    function onClickCreate(event) {
        setIsCreateMenuOpen(false);

        history.push(
            getEntityPath(entityType, undefined, {
                ...queryObject,
                action: 'create',
                type: event?.target?.value,
            })
        );
    }

    function onClickEdit() {
        history.push(getEntityPath(entityType, entityId, { ...queryObject, action: 'edit' }));
    }

    function onClickCancel() {
        dispatch(authActions.setSaveAuthProviderStatus(null));

        // The entityId is undefined for create and defined for update.
        history.push(getEntityPath(entityType, entityId, { ...queryObject, action: undefined }));
    }

    const selectedAuthProvider = authProviders.find(({ id }) => id === entityId);
    const hasAction = Boolean(action);
    const isList = typeof entityId !== 'string' && !hasAction;

    // if user elected to ignore a save error, don't pester them if they return to the form
    if (isList) {
        dispatch(authActions.setSaveAuthProviderStatus(null));
    }

    const dropdownItems = availableProviderTypes.map((curr) => (
        <DropdownItem key={curr.value} value={curr.value} component="button">
            {curr.label}
        </DropdownItem>
    ));

    return (
        <>
            <AccessControlPageTitle entityType={entityType} isList={isList} />
            <AccessControlHeading
                entityType={entityType}
                entityName={action === 'create' ? 'Add auth provider' : selectedAuthProvider?.name}
                isDisabled={hasAction}
                isList={isList}
            />
            <AccessControlNav entityType={entityType} />
            <AccessControlDescription>
                Configure authentication providers and rules to assign roles to users
            </AccessControlDescription>
            {isFetchingAuthProviders || isFetchingRoles ? (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            ) : isList ? (
                <>
                    <Toolbar inset={{ default: 'insetNone' }}>
                        <ToolbarContent>
                            <ToolbarItem>
                                <Title headingLevel="h2">Auth providers</Title>
                            </ToolbarItem>
                            <ToolbarItem>
                                <Badge isRead>{authProvidersWithRules.length}</Badge>
                            </ToolbarItem>
                            {hasWriteAccess && (
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
                            )}
                        </ToolbarContent>
                    </Toolbar>
                    {authProvidersWithRules.length === 0 && (
                        <ACSEmptyState title="No auth providers" headingLevel="h3">
                            Please add one.
                        </ACSEmptyState>
                    )}
                    {authProvidersWithRules.length > 0 && (
                        <AuthProvidersList
                            entityId={entityId}
                            authProviders={authProvidersWithRules}
                        />
                    )}
                </>
            ) : typeof entityId === 'string' && !selectedAuthProvider ? (
                <NotFoundMessage
                    title="Auth provider does not exist"
                    message={`Auth provider id: ${entityId}`}
                    actionText="Auth providers"
                    url={getEntityPath(entityType)}
                />
            ) : (
                <AuthProviderForm
                    isActionable={hasWriteAccess}
                    action={action}
                    selectedAuthProvider={selectedAuthProvider ?? getNewAuthProviderObj(type)}
                    onClickCancel={onClickCancel}
                    onClickEdit={onClickEdit}
                />
            )}
        </>
    );
}

export default AuthProviders;
