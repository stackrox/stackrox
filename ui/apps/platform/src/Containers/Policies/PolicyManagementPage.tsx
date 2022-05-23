import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    policiesPath,
    policyManagementBasePath,
    deprecatedPoliciesBasePath,
    deprecatedPoliciesPath,
} from 'routePaths';
import PoliciesPage from './PoliciesPage';

function PolicyManagementPage() {
    return (
        <Switch>
            <Redirect exact from={policyManagementBasePath} to={policiesPath} />
            <Redirect exact from={deprecatedPoliciesBasePath} to={policiesPath} />
            <Redirect from={deprecatedPoliciesPath} to={policiesPath} />
            <Route path={policiesPath}>
                <PoliciesPage />
            </Route>
        </Switch>
    );
}

export default PolicyManagementPage;
