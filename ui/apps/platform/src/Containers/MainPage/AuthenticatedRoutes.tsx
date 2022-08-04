import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { Redirect, useLocation } from 'react-router-dom';
import { createStructuredSelector } from 'reselect';

import LoadingSection from 'Components/PatternFly/LoadingSection';
import { selectors } from 'reducers';
import { AUTH_STATUS } from 'reducers/auth';

import MainPage from './MainPage';

const authStatusSelector = createStructuredSelector<{ authStatus: string }>({
    authStatus: selectors.getAuthStatus,
});

function AuthenticatedRoutes(): ReactElement {
    const { authStatus } = useSelector(authStatusSelector);
    const location = useLocation();

    switch (authStatus) {
        case AUTH_STATUS.LOADING:
            return <LoadingSection message="Authenticating..." />;

        case AUTH_STATUS.LOGGED_IN:
        case AUTH_STATUS.ANONYMOUS_ACCESS:
            return <MainPage />;

        case AUTH_STATUS.LOGGED_OUT:
        case AUTH_STATUS.AUTH_PROVIDERS_LOADING_ERROR:
        case AUTH_STATUS.LOGIN_AUTH_PROVIDERS_LOADING_ERROR:
            return (
                <Redirect
                    to={{
                        pathname: '/login',
                        state: { from: location.pathname },
                    }}
                />
            );

        default:
            throw new Error(`Unknown auth status: ${authStatus}`);
    }
}

export default AuthenticatedRoutes;
