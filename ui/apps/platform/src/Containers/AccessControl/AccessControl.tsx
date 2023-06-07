import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import { Alert, List, ListItem } from '@patternfly/react-core';

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
            <Alert
                isInline
                variant="warning"
                title={
                    <>
                        <p>The following permission resources have been replaced:</p>
                        <List>
                            <ListItem>
                                <b>Access</b> replaces{' '}
                                <b>AuthProvider, Group, Licenses, Role, and User</b>
                            </ListItem>
                            <ListItem>
                                <b>WorkflowAdministration</b> replaces{' '}
                                <b>Policy and VulnerabilityReports</b>
                            </ListItem>
                        </List>
                        <p>
                            For additional information on deprecation and required actions, please
                            consult the release notes.
                        </p>
                    </>
                }
            />
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
        </>
    );
}

export default AccessControl;
