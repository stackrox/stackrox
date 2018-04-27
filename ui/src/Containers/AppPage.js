import React from 'react';
import { Route, Switch, Redirect } from 'react-router-dom';

import ProtectedRoute from 'Components/ProtectedRoute';
import LoadingSection from 'Components/LoadingSection';
import MainPage from 'Containers/MainPage';
import LoginPage from 'Containers/Login/LoginPage';

const AppPage = () => (
    <Switch>
        <ProtectedRoute path="/main" component={MainPage} />
        <Route path="/login" component={LoginPage} />
        <Route path="/auth/response/oidc" component={LoadingSection} />
        <Redirect from="/" to="/main" />
    </Switch>
);

export default AppPage;
