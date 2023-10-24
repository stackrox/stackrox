/* eslint-disable no-nested-ternary */
import React, { ReactElement, useEffect, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { useHistory, useLocation, useParams } from 'react-router-dom';
import {
    Bullseye,
    Button,
    Dropdown,
    DropdownItem,
    DropdownPosition,
    DropdownToggle,
    PageSection,
    pluralize,
    Spinner,
    Title,
} from '@patternfly/react-core';
import { CaretDownIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import NotFoundMessage from 'Components/NotFoundMessage';
import { actions as authActions, types as authActionTypes } from 'reducers/auth';
import { actions as groupActions } from 'reducers/groups';
import { actions as inviteActions } from 'reducers/invite';
import { actions as roleActions, types as roleActionTypes } from 'reducers/roles';
import { AuthProvider } from 'services/AuthService';

import { getEntityPath, getQueryObject } from '../accessControlPaths';
import { mergeGroupsWithAuthProviders } from './authProviders.utils';

import AccessControlPageTitle from '../AccessControlPageTitle';

import AccessControlDescription from '../AccessControlDescription';
import AuthProviderForm from './AuthProviderForm';
import AuthProvidersList from './AuthProvidersList';
import AccessControlBreadcrumbs from '../AccessControlBreadcrumbs';
import AccessControlHeading from '../AccessControlHeading';
import AccessControlHeaderActionBar from '../AccessControlHeaderActionBar';
import usePermissions from '../../../hooks/usePermissions';

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
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForPage = hasReadWriteAccess('Access');
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
        availableProviderTypes,
    } = useSelector(authProviderState);

    const authProvidersWithRules = mergeGroupsWithAuthProviders(authProviders, groups);

    useEffect(() => {
        dispatch(authActions.fetchAuthProviders.request());
        dispatch(roleActions.fetchRoles.request());
        dispatch(groupActions.fetchGroups.request());
    }, [dispatch]);

    function onToggleCreateMenu(isOpen) {
        setIsCreateMenuOpen(isOpen);
    }

    function onClickInviteUsers() {
        dispatch(inviteActions.setInviteModalVisibility(true));
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

    function getProviderLabel(): string {
        const provider = availableProviderTypes.find(({ value }) => value === type) ?? {};
        return (provider.label as string) ?? 'auth';
    }

    const selectedAuthProvider = authProviders.find(({ id }) => id === entityId);
    const hasAction = Boolean(action);
    const isList = typeof entityId !== 'string' && !hasAction;

    // if user elected to ignore a save error, don't pester them if they return to the form
    if (isList) {
        dispatch(authActions.setSaveAuthProviderStatus(null));
    }

    const dropdownItems = availableProviderTypes.map(({ value, label }) => (
        <DropdownItem key={value} value={value} component="button">
            {label}
        </DropdownItem>
    ));

    return (
        <>
            <AccessControlPageTitle entityType={entityType} isList={isList} />
            {isList ? (
                <>
                    <AccessControlHeading entityType={entityType} />
                    <AccessControlHeaderActionBar
                        displayComponent={
                            <AccessControlDescription>
                                Configure authentication providers and rules to assign roles to
                                users
                            </AccessControlDescription>
                        }
                        inviteComponent={
                            hasWriteAccessForPage && (
                                <Button variant="secondary" onClick={onClickInviteUsers}>
                                    Invite users
                                </Button>
                            )
                        }
                        actionComponent={
                            hasWriteAccessForPage && (
                                <Dropdown
                                    className="auth-provider-dropdown pf-u-ml-md"
                                    onSelect={onClickCreate}
                                    position={DropdownPosition.right}
                                    toggle={
                                        <DropdownToggle
                                            onToggle={onToggleCreateMenu}
                                            toggleIndicator={CaretDownIcon}
                                            isPrimary
                                        >
                                            Create auth provider
                                        </DropdownToggle>
                                    }
                                    isOpen={isCreateMenuOpen}
                                    dropdownItems={dropdownItems}
                                />
                            )
                        }
                    />
                </>
            ) : (
                <AccessControlBreadcrumbs
                    entityType={entityType}
                    entityName={
                        action === 'create'
                            ? `Create ${getProviderLabel()} provider`
                            : selectedAuthProvider?.name
                    }
                />
            )}
            <PageSection variant={isList ? 'default' : 'light'}>
                {isFetchingAuthProviders || isFetchingRoles ? (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                ) : isList ? (
                    <PageSection variant="light">
                        <Title headingLevel="h2">
                            {pluralize(authProvidersWithRules.length, 'result')} found
                        </Title>
                        {authProvidersWithRules.length === 0 && (
                            <EmptyStateTemplate title="No auth providers" headingLevel="h3">
                                Please add one.
                            </EmptyStateTemplate>
                        )}
                        {authProvidersWithRules.length > 0 && (
                            <AuthProvidersList
                                entityId={entityId}
                                authProviders={authProvidersWithRules}
                            />
                        )}
                    </PageSection>
                ) : typeof entityId === 'string' && !selectedAuthProvider ? (
                    <NotFoundMessage
                        title="Auth provider does not exist"
                        message={`Auth provider id: ${entityId}`}
                        actionText="Auth providers"
                        url={getEntityPath(entityType)}
                    />
                ) : (
                    <AuthProviderForm
                        isActionable={hasWriteAccessForPage}
                        action={action}
                        selectedAuthProvider={selectedAuthProvider ?? getNewAuthProviderObj(type)}
                        onClickCancel={onClickCancel}
                        onClickEdit={onClickEdit}
                    />
                )}
            </PageSection>
        </>
    );
}

export default AuthProviders;
