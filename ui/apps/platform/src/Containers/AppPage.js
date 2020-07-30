import React from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import {
    mainPath,
    loginPath,
    testLoginResultsPath,
    licenseStartUpPath,
    authResponsePrefix,
} from 'routePaths';
import ProtectedRoute from 'Components/ProtectedRoute';
import LoadingSection from 'Components/LoadingSection';
import MainPage from 'Containers/MainPage';
import LoginPage from 'Containers/Login/LoginPage';
import TestLoginResultsPage from 'Containers/Login/TestLoginResultsPage';
import LicenseStartUpScreen from 'Containers/License/StartUpScreen';

const AppPage = () => (
    <Switch>
        <ProtectedRoute path={mainPath} component={MainPage} />
        <Route path={licenseStartUpPath} component={LicenseStartUpScreen} />
        <Route path={loginPath} component={LoginPage} />
        <Route path={testLoginResultsPath} component={TestLoginResultsPage} />
        <Route path={authResponsePrefix} component={LoadingSection} />
        <Redirect from="/" to={mainPath} />
    </Switch>
);

export default AppPage;
