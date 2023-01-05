import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import { Alert, List, ListItem } from '@patternfly/react-core';
import { accessControlBasePath, accessControlPath, getEntityPath } from './accessControlPaths';

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
                                <b>AuthProvider, Group, Licenses, and User</b>
                            </ListItem>
                            <ListItem>
                                <b>DeploymentExtension</b> replaces{' '}
                                <b>Indicator, NetworkBaseline, ProcessWhitelist, and Risk</b>
                            </ListItem>
                            <ListItem>
                                <b>Integration</b> replaces{' '}
                                <b>
                                    APIToken, BackupPlugins, ImageIntegration, Notifier, and
                                    SignatureIntegration
                                </b>
                            </ListItem>
                            <ListItem>
                                <b>Image</b> now also covers <b>ImageComponent</b>
                            </ListItem>
                            <ListItem>
                                <b>Cluster</b> now also covers <b>ClusterCVE</b>
                            </ListItem>
                        </List>

                        <p>
                            The following permission resources will be replaced in the upcoming
                            versions:
                        </p>
                        <List>
                            <ListItem>
                                <b>Administration</b> will replace{' '}
                                <b>
                                    AllComments, Config, DebugLogs, NetworkGraphConfig, ProbeUpload,
                                    ScannerBundle, ScannerDefinitions, SensorUpgradeConfig, and
                                    ServiceIdentity
                                </b>
                            </ListItem>
                            <ListItem>
                                <b>Compliance</b> will replace <b>ComplianceRuns</b>
                            </ListItem>
                        </List>
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
