import React from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import { mainPath, loginPath, authResponsePrefix } from 'routePaths';
import ProtectedRoute from 'Components/ProtectedRoute';
import LoadingSection from 'Components/LoadingSection';
import MainPage from 'Containers/MainPage';
import LoginPage from 'Containers/Login/LoginPage';

const AppPage = () => (
    <Switch>
        <ProtectedRoute path={mainPath} component={MainPage} />
        <Route path={loginPath} component={LoginPage} />
        <Route path={authResponsePrefix} component={LoadingSection} />
        <Redirect from="/" to={mainPath} />
    </Switch>
);

export default AppPage;
