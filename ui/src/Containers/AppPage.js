import React from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import ProtectedRoute from 'Components/ProtectedRoute';
import AuthRedirectRoute from 'Components/AuthRedirectRoute';
import Auth from 'Containers/Auth';
import MainPage from 'Containers/MainPage';
import LoginPage from 'Containers/Login/LoginPage';

const AppPage = () => (
    <Auth>
        <Switch>
            <ProtectedRoute path="/main" component={MainPage} />
            <Route path="/login" component={LoginPage} />
            <AuthRedirectRoute path="/auth/response/oidc" />
            <Redirect from="/" to="/main" />
        </Switch>
    </Auth>
);

export default AppPage;
