import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { ClipLoader } from 'react-spinners';

import Select from 'Components/ReactSelect';
import UnreachableWarning from 'Containers/UnreachableWarning';

import logoPlatform from 'images/logo-platform.svg';

import { AUTH_STATUS } from 'reducers/auth';
import { selectors } from 'reducers';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';

import AppWrapper from '../AppWrapper';
import LoginNotice from './LoginNotice';

class LoginPage extends Component {
    static propTypes = {
        authStatus: PropTypes.oneOf(Object.keys(AUTH_STATUS).map(key => AUTH_STATUS[key]))
            .isRequired,
        authProviders: PropTypes.arrayOf(PropTypes.object).isRequired,
        authProviderResponse: PropTypes.shape({
            error: PropTypes.string,
            error_description: PropTypes.string,
            error_uri: PropTypes.string
        }).isRequired
    };

    constructor(props) {
        super(props);
        const { authProviders } = props;
        this.state = {
            selectedAuthProviderId: authProviders.length > 0 ? authProviders[0].id : null,
            showAuthError: true
        };
    }

    componentWillReceiveProps(nextProps) {
        // pre-select first auth provider
        if (!this.state.selectedAuthProviderId && nextProps.authProviders.length > 0) {
            this.setState({ selectedAuthProviderId: nextProps.authProviders[0].id });
        }
    }

    onAuthProviderSelected = id => this.setState({ selectedAuthProviderId: id });

    login = () => {
        const { selectedAuthProviderId } = this.state;
        const authProvider = this.props.authProviders.find(ap => ap.id === selectedAuthProviderId);
        if (!authProvider) return;
        window.location = authProvider.loginUrl; // redirect to external URL, so no react-router
    };

    dismissAuthError = () => this.setState({ showAuthError: false });

    renderAuthError = () => {
        const fg = 'alert-800';
        const bg = 'alert-200';
        const color = `text-${fg} bg-${bg}`;
        const { authProviderResponse } = this.props;
        const closeButton = (
            <div>
                <Tooltip placement="top" overlay={<div>Dismiss</div>}>
                    <button
                        type="button"
                        className={`flex p-1 text-center text-sm items-center p-2 ${color} hover:bg-${fg} hover:text-${bg} border-l border-${fg}`}
                        onClick={this.dismissAuthError}
                        data-test-id="dismiss"
                    >
                        <Icon.X className="h-4" />
                    </button>
                </Tooltip>
            </div>
        );
        if (this.state.showAuthError && authProviderResponse.error) {
            const errorKey = authProviderResponse.error.replace('_', ' ');
            const errorMsg = authProviderResponse.error_description || '';
            const errorLink = (url =>
                url ? (
                    <span>
                        (
                        <a className={`${color}`} href={url}>
                            more info
                        </a>
                        )
                    </span>
                ) : (
                    []
                ))(authProviderResponse.error_uri);
            return (
                <div className={`flex items-center font-sans w-full text-center h-full ${color}`}>
                    <span className="w-full">
                        <span className="capitalize">{errorKey}</span>. {errorMsg} {errorLink}
                    </span>
                    {closeButton}
                </div>
            );
        }
        return null;
    };

    renderAuthProviders = () => {
        const { authStatus, authProviders } = this.props;
        if (
            authStatus === AUTH_STATUS.LOADING ||
            authStatus === AUTH_STATUS.LOGGED_IN ||
            authStatus === AUTH_STATUS.ANONYMOUS_ACCESS
        ) {
            return null;
        }
        const options = authProviders
            .filter(obj => obj.enabled)
            .map(authProvider => ({ label: authProvider.name, value: authProvider.id }));
        const { selectedAuthProviderId } = this.state;
        return (
            <div className="py-8 items-center w-2/3">
                <div className="text-primary-700 font-700 pb-3">Select an auth provider</div>
                <Select
                    value={selectedAuthProviderId}
                    isClearable={false}
                    isDisabled={authProviders.length === 1}
                    onChange={this.onAuthProviderSelected}
                    options={options}
                />
            </div>
        );
    };

    renderLoginButton = () => {
        const { authStatus } = this.props;
        if (authStatus === AUTH_STATUS.LOADING) {
            return (
                <div className="border-t border-base-300 p-6 w-full text-center">
                    <button
                        type="button"
                        className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 uppercase text-center tracking-wide"
                    >
                        <ClipLoader color="white" loading size={15} />
                    </button>
                </div>
            );
        }
        if (authStatus === AUTH_STATUS.LOGGED_IN || authStatus === AUTH_STATUS.ANONYMOUS_ACCESS) {
            return (
                <div className="border-t border-base-300 p-8 w-full text-center">
                    <Link
                        className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 uppercase text-center tracking-wide no-underline"
                        to="/main/dashboard"
                    >
                        Go to Dashboard
                    </Link>
                </div>
            );
        }
        return (
            <div className="border-t border-base-300 p-6 w-full text-center">
                <button
                    type="button"
                    className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 uppercase text-center tracking-wide"
                    onClick={this.login}
                >
                    Login
                </button>
            </div>
        );
    };

    render() {
        const isDarkMode = localStorage.getItem('isDarkMode') === 'true';

        return (
            <AppWrapper>
                <section
                    className={`flex flex-col items-center justify-center h-full py-5 ${
                        isDarkMode ? 'bg-base-300' : 'bg-primary-800'
                    } `}
                >
                    <div className="flex flex-col items-center bg-base-100 w-2/5 md:w-3/5 xl:w-2/5 relative">
                        {this.renderAuthError()}
                    </div>
                    <div className="flex flex-col items-center justify-center bg-base-100 w-2/5 md:w-3/5 xl:w-2/5 relative login-bg">
                        <UnreachableWarning />
                        <div className="login-border-t h-1 w-full" />
                        <div className="flex flex-col items-center justify-center w-full">
                            <img className="h-40 h-40 py-6" src={logoPlatform} alt="StackRox" />
                            {this.renderAuthProviders()}
                        </div>
                        <LoginNotice />
                        {this.renderLoginButton()}
                    </div>
                </section>
            </AppWrapper>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    authStatus: selectors.getAuthStatus,
    authProviderResponse: selectors.getAuthProviderError
});

export default connect(mapStateToProps)(LoginPage);
