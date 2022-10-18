import React, { ReactElement, useEffect } from 'react';
import { Route, Switch } from 'react-router-dom';
import { useDispatch } from 'react-redux';

import { loginPath, testLoginResultsPath, authResponsePrefix } from 'routePaths';
import LoadingSection from 'Components/PatternFly/LoadingSection';
import AuthenticatedRoutes from 'Containers/MainPage/AuthenticatedRoutes';
import LoginPage from 'Containers/Login/LoginPage';
import TestLoginResultsPage from 'Containers/Login/TestLoginResultsPage';
import AppPageTitle from 'Containers/AppPageTitle';
import AppPageFavicon from 'Containers/AppPageFavicon';
import * as service from 'services/FeatureFlagsService';
import { fetchPublicConfig as fetchPublicConfigService } from 'services/SystemConfigService';
import { actions as featureFlagActions } from 'reducers/featureFlags';
import { actions as publicConfigActions } from 'reducers/systemConfig';

function AppPage(): ReactElement {
    const dispatch = useDispatch();
    useEffect(() => {
        const fetchFlags = async () => {
            try {
                const result = await service.fetchFeatureFlags();
                dispatch(featureFlagActions.fetchFeatureFlags.success(result.response));
            } catch (e) {
                const error = e as Error;
                dispatch(featureFlagActions.fetchFeatureFlags.failure(error));
            }
        };

        const fetchPublicConfig = async () => {
            try {
                const result = await fetchPublicConfigService();
                dispatch(publicConfigActions.fetchPublicConfig.success(result.response));
            } catch (e) {
                const error = e as Error;
                dispatch(publicConfigActions.fetchPublicConfig.failure(error));
            }
        };

        dispatch(featureFlagActions.fetchFeatureFlags.request());
        fetchFlags().catch(() => {
            throw Error('generic error');
        });

        dispatch(publicConfigActions.fetchPublicConfig.request());
        // eslint-disable-next-line no-void
        void fetchPublicConfig();
    }, [dispatch]);

    return (
        <>
            <AppPageTitle />
            <AppPageFavicon />
            <Switch>
                <Route path={loginPath} component={LoginPage} />
                <Route path={testLoginResultsPath} component={TestLoginResultsPage} />
                <Route path={authResponsePrefix} component={LoadingSection} />
                <Route component={AuthenticatedRoutes} />
            </Switch>
        </>
    );
}

export default AppPage;
