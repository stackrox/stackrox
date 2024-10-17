import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    policiesBasePath,
    policiesPath,
    policyManagementBasePath,
    policyCategoriesPath,
} from 'routePaths';
import PoliciesPage from 'Containers/Policies/PoliciesPage';
import PolicyCategoriesPage from 'Containers/PolicyCategories/PolicyCategoriesPage';

function PolicyManagementPage() {
    return (
        <Switch>
            <Route
                exact
                path={policyManagementBasePath}
                render={() => <Redirect to={policiesBasePath} />}
            />
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
