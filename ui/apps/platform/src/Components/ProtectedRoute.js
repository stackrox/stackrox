import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { Route, Redirect } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { AUTH_STATUS } from 'reducers/auth';
import { getHasReadPermission } from 'reducers/roles';
import LoadingSection from 'Components/PatternFly/LoadingSection';

class ProtectedRoute extends Component {
    static propTypes = {
        path: PropTypes.string.isRequired,
        component: PropTypes.elementType.isRequired,
        authStatus: PropTypes.oneOf(Object.keys(AUTH_STATUS).map((key) => AUTH_STATUS[key]))
            .isRequired,
        location: ReactRouterPropTypes.location, // provided by Switch but omit isRequired because TypeScript does not know
        devOnly: PropTypes.bool,
        requiredPermission: PropTypes.string,
        userRolePermissions: PropTypes.shape({
            resourceToAccess: PropTypes.shape({}),
        }),
        featureFlagEnabled: PropTypes.bool,
    };

    static defaultProps = {
        location: { pathname: '' }, // see comment above
        devOnly: false,
        requiredPermission: null,
        userRolePermissions: null,
        featureFlagEnabled: true,
    };

    renderRoute = (props) => {
        const { component: LocationComponent, authStatus, location } = this.props;

        switch (authStatus) {
            case AUTH_STATUS.LOADING:
                return <LoadingSection message="Authenticating..." />;
            case AUTH_STATUS.LOGGED_IN:
            case AUTH_STATUS.ANONYMOUS_ACCESS:
                return <LocationComponent {...props} />;
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
    };

    render() {
        const {
            component,
            authStatus,
            devOnly,
            requiredPermission,
            userRolePermissions,
            featureFlagEnabled,
            ...rest
        } = this.props;

        if (
            !featureFlagEnabled ||
            (devOnly && process.env.NODE_ENV !== 'development') ||
            (requiredPermission && !getHasReadPermission(requiredPermission, userRolePermissions))
        ) {
            return <Redirect to="/" />;
        }

        return <Route {...rest} render={this.renderRoute} />;
    }
}

const mapStateToProps = createStructuredSelector({
    authStatus: selectors.getAuthStatus,
    userRolePermissions: selectors.getUserRolePermissions,
});

export default connect(mapStateToProps)(ProtectedRoute);
