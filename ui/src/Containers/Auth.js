import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { ClipLoader } from 'react-spinners';
import axios from 'axios';

import AuthService from 'services/AuthService';

const excludedUrls = ['/login', '/auth/response/oidc'];

class Auth extends Component {
    static propTypes = {
        location: ReactRouterPropTypes.location.isRequired,
        children: PropTypes.node.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            showPage: false
        };
    }

    componentDidMount() {
        AuthService.updateAuthProviders()
            .then(() => {
                if (
                    AuthService.getAuthProviders().length === 0 ||
                    excludedUrls.includes(this.props.location.pathname)
                )
                    this.setState({ showPage: true });
                else {
                    axios
                        .get('/v1/auth/status')
                        .then(() => {
                            this.setState({ showPage: true });
                        })
                        .catch(error => console.log(error));
                }
            })
            .catch(error => console.log(error));
    }

    render() {
        if (!this.state.showPage) {
            return (
                <section className="flex flex-col items-center justify-center h-full login-bg">
                    <ClipLoader color="white" loading size={20} />
                    <div className="text-lg font-sans text-white tracking-wide mt-4">
                        Loading...
                    </div>
                </section>
            );
        }
        return this.props.children;
    }
}

export default withRouter(Auth);
