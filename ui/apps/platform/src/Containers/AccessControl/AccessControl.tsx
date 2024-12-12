import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import { accessControlBasePath, accessControlPath } from 'routePaths';

import { getEntityPath } from './accessControlPaths';

import AccessControlRouteNotFound from './AccessControlRouteNotFound';
import AccessScopes from './AccessScopes/AccessScopes';
import AuthProviders from './AuthProviders/AuthProviders';
import PermissionSets from './PermissionSets/PermissionSets';
import Roles from './Roles/Roles';

const paramId = ':entityId?';

function AccessControl(): ReactElement {
    return (
        <>
            <Switch>
                <Route
                    exact
                    path={accessControlBasePath}
                    render={() => <Redirect to={getEntityPath('AUTH_PROVIDER')} />}
                />
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
        </>
    );
}

export default AccessControl;
