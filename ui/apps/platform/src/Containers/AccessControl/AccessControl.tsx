import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import usePermissions from 'hooks/usePermissions';

import { Alert } from '@patternfly/react-core';
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
    const hasReadAccessForAccessControlPages = hasReadAccess('Access');

    return (
        <>
            <Alert
                isInline
                variant="warning"
                title={
                    <>
                        <p>The following permission resources have been replaced:</p>
                        <ul>
                            <li>
                                <b>Access</b> replaces{' '}
                                <b>AuthProvider, Group, Licenses, and User</b>
                            </li>
                            <li>
                                <b>DeploymentExtension</b> replaces{' '}
                                <b>Indicator, NetworkBaseline, ProcessWhitelist, and Risk</b>
                            </li>
                            <li>
                                <b>Integration</b> replaces{' '}
                                <b>
                                    APIToken, BackupPlugins, ImageIntegration, Notifier, and
                                    SignatureIntegration
                                </b>
                            </li>
                            <li>
                                <b>Image</b> now also covers <b>ImageComponent</b>
                            </li>
                        </ul>

                        <p>
                            The following permission resources will be replaced in the upcoming
                            versions:
                        </p>
                        <ul>
                            <li>
                                <b>Administration</b> will replace{' '}
                                <b>
                                    AllComments, Config, DebugLogs, NetworkGraphConfig, ProbeUpload,
                                    ScannerDefinitions, SensorUpgradeConfig, and ServiceIdentity
                                </b>
                            </li>
                            <li>
                                <b>Compliance</b> will replace <b>ComplianceRuns</b>
                            </li>
                            <li>
                                <b>Cluster</b> will replace <b>ClusterCVE</b>
                            </li>
                        </ul>
                    </>
                }
            />
            {hasReadAccessForAccessControlPages ? (
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
