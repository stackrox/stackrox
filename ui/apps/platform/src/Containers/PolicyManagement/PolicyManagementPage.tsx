import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import { policiesPath, policyManagementBasePath, policyCategoriesPath } from 'routePaths';
import PoliciesPage from 'Containers/Policies/PoliciesPage';
import PolicyCategoriesPage from 'Containers/PolicyCategories/PolicyCategoriesPage';

function PolicyManagementPage() {
    return (
        <Switch>
            <Redirect exact from={policyManagementBasePath} to={policiesPath} />
            <Route path={policiesPath}>
                <PoliciesPage />
            </Route>
            <Route path={policyCategoriesPath}>
                <PolicyCategoriesPage />
            </Route>
        </Switch>
    );
}

export default PolicyManagementPage;
