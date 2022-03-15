import React from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import { mainPath, loginPath, testLoginResultsPath, authResponsePrefix } from 'routePaths';
import ProtectedRoute from 'Components/ProtectedRoute';
import LoadingSection from 'Components/PatternFly/LoadingSection';
import MainPage from 'Containers/MainPage';
import LoginPage from 'Containers/Login/LoginPage';
import TestLoginResultsPage from 'Containers/Login/TestLoginResultsPage';
import AppPageTitle from 'Containers/AppPageTitle';
import AppPageFavicon from 'Containers/AppPageFavicon';

const AppPage = () => {
    return (
        <>
            <AppPageTitle />
            <AppPageFavicon />
            <Switch>
                <ProtectedRoute path={mainPath} component={MainPage} />
                <Route path={loginPath} component={LoginPage} />
                <Route path={testLoginResultsPath} component={TestLoginResultsPage} />
                <Route path={authResponsePrefix} component={LoadingSection} />
                <Redirect from="/" to={mainPath} />
            </Switch>
        </>
    );
};

export default AppPage;
