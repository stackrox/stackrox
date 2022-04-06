import React, { ReactElement } from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import { mainPath, loginPath, testLoginResultsPath, authResponsePrefix } from 'routePaths';
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
                <Route path={mainPath} component={AuthenticatedRoutes} />
                <Route path={loginPath} component={LoginPage} />
                <Route path={testLoginResultsPath} component={TestLoginResultsPage} />
                <Route path={authResponsePrefix} component={LoadingSection} />
                <Redirect from="/" to={mainPath} />
            </Switch>
        </>
    );
}

export default AppPage;
