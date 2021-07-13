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
import { actions as groupActions } from 'reducers/groups';
import { actions as roleActions, types as roleActionTypes } from 'reducers/roles';
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
    const { authProviders, groups, isFetchingAuthProviders, isFetchingRoles } = useSelector(
        authProviderState
    );

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

    return (
        <>
            <AccessControlPageTitle entityType={entityType} isEntity={isExpanded} />
            <AccessControlHeading
                entityType={entityType}
                entityName={
                    selectedAuthProvider &&
                    (action === 'create' ? 'Add auth provider' : selectedAuthProvider.name)
                }
                isDisabled={hasAction}
            />
            <AccessControlNav entityType={entityType} />
            <AccessControlDescription>
                Configure authentication providers and rules to assign roles to users
            </AccessControlDescription>
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
                    onClickCancel={onClickCancel}
                    onClickEdit={onClickEdit}
                />
            )}
            {!isFetchingAuthProviders && !isFetchingRoles && !isExpanded && (
                <>
                    <Toolbar inset={{ default: 'insetNone' }}>
                        <ToolbarContent>
                            <ToolbarItem>
                                <Title headingLevel="h2">Auth providers</Title>
                            </ToolbarItem>
                            <ToolbarItem>
                                <Badge isRead>{authProvidersWithRules.length}</Badge>
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
                    <AuthProvidersList entityId={entityId} authProviders={authProvidersWithRules} />
                </>
            )}
        </>
    );
}

export default AuthProviders;
