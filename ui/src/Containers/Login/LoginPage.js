import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { ClipLoader } from 'react-spinners';

import Select from 'Components/ReactSelect';

import logoPrevent from 'images/logo-prevent.svg';

import { AUTH_STATUS } from 'reducers/auth';
import { selectors } from 'reducers';

class LoginPage extends Component {
    static propTypes = {
        authStatus: PropTypes.oneOf(Object.keys(AUTH_STATUS).map(key => AUTH_STATUS[key]))
            .isRequired,
        authProviders: PropTypes.arrayOf(PropTypes.object).isRequired
    };

    constructor(props) {
        super(props);
        const { authProviders } = props;
        this.state = {
            selectedAuthProviderId: authProviders.length > 0 ? authProviders[0].id : null
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
        window.location = authProvider.loginUrl; // redirect to external URL, so no react-router
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
        return (
            <section className="flex flex-col items-center justify-center h-full bg-primary-800">
                <div className="flex flex-col items-center justify-center bg-base-100 w-2/5 w-4/5 md:w-3/5 xl:w-2/5 relative login-bg">
                    <div className="login-border-t h-1 w-full" />
                    <div className="flex flex-col items-center justify-center w-full">
                        <img className="h-40 h-40 py-6" src={logoPrevent} alt="StackRox" />
                        {this.renderAuthProviders()}
                    </div>
                    {this.renderLoginButton()}
                </div>
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    authStatus: selectors.getAuthStatus
});

export default connect(mapStateToProps)(LoginPage);
