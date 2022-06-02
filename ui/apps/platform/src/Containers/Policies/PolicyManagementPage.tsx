import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import { policiesPath, policyManagementBasePath, deprecatedPoliciesBasePath } from 'routePaths';
import PoliciesPage from './PoliciesPage';

function PolicyManagementPage() {
    return (
        <Switch>
            <Redirect exact from={policyManagementBasePath} to={policiesPath} />
            <Redirect exact from={deprecatedPoliciesBasePath} to={policiesPath} />
            <Route path={policiesPath}>
                <PoliciesPage />
            </Route>
        </Switch>
    );
}

export default PolicyManagementPage;
