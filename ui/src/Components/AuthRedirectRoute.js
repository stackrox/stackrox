import React, { Component } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import queryString from 'query-string';
import { ClipLoader } from 'react-spinners';

import AuthService from 'services/AuthService';

class AuthRedirectRoute extends Component {
    static propTypes = {
        location: ReactRouterPropTypes.location.isRequired,
        history: ReactRouterPropTypes.history.isRequired
    };

    componentDidMount() {
        const { location } = this.props;
        const accessToken = queryString.parse(location.hash).access_token;
        AuthService.login(accessToken);
        this.props.history.push('/');
    }

    render() {
        return (
            <section className="flex flex-col items-center justify-center h-full bg-primary-600">
                <ClipLoader color="white" loading size={20} />
                <div className="text-lg font-sans text-white tracking-wide mt-4">
                    Redirecting...
                </div>
            </section>
        );
    }
}

export default withRouter(AuthRedirectRoute);
