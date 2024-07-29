import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import {
    loginPath,
    testLoginResultsPath,
    authResponsePrefix,
    authorizeRoxctlPath,
} from 'routePaths';
import LoadingSection from 'Components/PatternFly/LoadingSection';
import AuthenticatedRoutes from 'Containers/MainPage/AuthenticatedRoutes';
import LoginPage from 'Containers/Login/LoginPage';
import TestLoginResultsPage from 'Containers/Login/TestLoginResultsPage';
import AppPageTitle from 'Containers/AppPageTitle';
import AppPageFavicon from 'Containers/AppPageFavicon';

function AppPage(): ReactElement {
    return (
        <>
            <AppPageTitle />
            <AppPageFavicon />
            <Switch>
                <Route path={loginPath}>
                    <LoginPage authProviders={undefined} />
                </Route>
                <Route path={authorizeRoxctlPath}>
                    <LoginPage authorizeRoxctlMode authProviders={undefined} />
                </Route>
                <Route path={testLoginResultsPath}>
                    <TestLoginResultsPage />
                </Route>
                <Route path={authResponsePrefix}>
                    <LoadingSection />
                </Route>
                <Route>
                    <AuthenticatedRoutes />
                </Route>
            </Switch>
        </>
    );
}

export default AppPage;
