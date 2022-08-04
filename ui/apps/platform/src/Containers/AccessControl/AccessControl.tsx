import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import usePermissions from 'hooks/usePermissions';

import { accessControlBasePath, accessControlPath, getEntityPath } from './accessControlPaths';

import AccessControlNoPermission from './AccessControlNoPermission';
import AccessControlRouteNotFound from './AccessControlRouteNotFound';
import AccessScopes from './AccessScopes/AccessScopes';
import AuthProviders from './AuthProviders/AuthProviders';
import PermissionSets from './PermissionSets/PermissionSets';
import Roles from './Roles/Roles';

const paramId = ':entityId?';

function AccessControl(): ReactElement {
    // TODO is read access required for all routes in improved Access Control?
    // TODO Is write access required anywhere in classic Access Control?
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAuthProvider = hasReadAccess('AuthProvider');

    return (
        <>
            {hasReadAccessForAuthProvider ? (
                <Switch>
                    <Route exact path={accessControlBasePath}>
                        <Redirect to={getEntityPath('AUTH_PROVIDER')} />
                    </Route>
                    <Route path={accessControlPath}>
                        <Switch>
                            <Route path={getEntityPath('AUTH_PROVIDER', paramId)}>
                                <AuthProviders />
                            </Route>
                            <Route path={getEntityPath('ROLE', paramId)}>
                                <Roles />
                            </Route>
                            <Route path={getEntityPath('PERMISSION_SET', paramId)}>
                                <PermissionSets />
                            </Route>
                            <Route path={getEntityPath('ACCESS_SCOPE', paramId)}>
                                <AccessScopes />
                            </Route>
                            <Route>
                                <AccessControlRouteNotFound />
                            </Route>
                        </Switch>
                    </Route>
                </Switch>
            ) : (
                <AccessControlNoPermission />
            )}
        </>
    );
}

export default AccessControl;
