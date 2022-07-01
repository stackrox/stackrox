import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import { policiesPath, policyManagementBasePath, policyCategoriesPath } from 'routePaths';
import useFeatureFlags from 'hooks/useFeatureFlags';
import PoliciesPage from 'Containers/Policies/PoliciesPage';
import PolicyCategoriesPage from 'Containers/PolicyCategories/PolicyCategoriesPage';

function PolicyManagementPage() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isPolicyCategoriesEnabled = isFeatureFlagEnabled('ROX_NEW_POLICY_CATEGORIES');
    return (
        <Switch>
            <Redirect exact from={policyManagementBasePath} to={policiesPath} />
            <Route path={policiesPath}>
                <PoliciesPage />
            </Route>
            {isPolicyCategoriesEnabled && (
                <Route path={policyCategoriesPath}>
                    <PolicyCategoriesPage />
                </Route>
            )}
        </Switch>
    );
}

export default PolicyManagementPage;
