import React from 'react';
import { connect } from 'react-redux';
import { Redirect, Route, Switch } from 'react-router-dom';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import { Alert, AlertVariant, PageSection, Title } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions, getHasReadPermission } from 'reducers/roles';

import { accessControlBasePath, accessControlPath, getEntityPath } from './accessControlPaths';

import AccessControlRouteNotFound from './AccessControlRouteNotFound';
import AccessScopes from './AccessScopes/AccessScopes';
import AuthProviders from './AuthProviders/AuthProviders';
import PermissionSets from './PermissionSets/PermissionSets';
import Roles from './Roles/Roles';

import './AccessControl.css';

const paramId = ':entityId?';

function AccessControl({ userRolePermissions }) {
    // TODO is read access required for all routes in improved Access Control?
    // TODO Is write access required anywhere in classic Access Control?
    const hasReadAccess = getHasReadPermission('AuthProvider', userRolePermissions);

    return (
        <PageSection variant="light" isFilled id="access-control">
            <Title headingLevel="h1" className="pf-u-pb-md">
                Access Control
            </Title>
            {hasReadAccess ? (
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
                <Alert
                    title="You do not have permission to view Access Control"
                    variant={AlertVariant.info}
                    isInline
                />
            )}
        </PageSection>
    );
}

AccessControl.propTypes = {
    userRolePermissions: PropTypes.shape({
        resourceToAccess: PropTypes.shape({ AuthProvider: PropTypes.string }),
    }).isRequired,
};

const mapStateToProps = createStructuredSelector({
    userRolePermissions: selectors.getUserRolePermissions,
});

const mapDispatchToProps = {
    fetchResources: actions.fetchResources.request,
};

export default connect(mapStateToProps, mapDispatchToProps)(AccessControl);
