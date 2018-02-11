import React from 'react';
import PropTypes from 'prop-types';
import { Route, Redirect } from 'react-router-dom';

import AuthService from 'Providers/AuthService';

const ProtectedRoute = ({ ...rest }) => {
    const to = {
        pathname: '/login'
    };
    if (AuthService.getAccessToken() || !AuthService.getAuthProviders().length) {
        return <Route {...rest} />;
    }
    return <Redirect to={to} />;
};

ProtectedRoute.propTypes = {
    component: PropTypes.oneOfType([PropTypes.element, PropTypes.func]).isRequired
};

export default ProtectedRoute;
