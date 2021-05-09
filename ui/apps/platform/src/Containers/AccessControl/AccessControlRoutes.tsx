import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import PageNotFound from 'Components/PageNotFound';
import useCaseTypes from 'constants/useCaseTypes'; // use case path segments

import { accessControlBasePath, getEntityPath } from './accessControlPaths';
import AccessScopesList from './AccessScopes/AccessScopesList';
import AuthProvidersList from './AuthProviders/AuthProvidersList';
import PermissionSetsList from './PermissionSets/PermissionSetsList';
import RolesList from './Roles/RolesList';

const entityIdParam = ':entityId?';

function AccessControlRoutes(): ReactElement {
    return (
        <Switch>
            <Route exact path={accessControlBasePath}>
                <Redirect to={getEntityPath('AUTH_PROVIDER')} />
            </Route>
            <Route path={getEntityPath('ACCESS_SCOPE', entityIdParam)}>
                <AccessScopesList />
            </Route>
            <Route path={getEntityPath('AUTH_PROVIDER', entityIdParam)}>
                <AuthProvidersList />
            </Route>
            <Route path={getEntityPath('PERMISSION_SET', entityIdParam)}>
                <PermissionSetsList />
            </Route>
            <Route path={getEntityPath('ROLE', entityIdParam)}>
                <RolesList />
            </Route>
            <Route>
                <PageNotFound useCase={useCaseTypes.ACCESS_CONTROL} />
            </Route>
        </Switch>
    );
}

export default AccessControlRoutes;
