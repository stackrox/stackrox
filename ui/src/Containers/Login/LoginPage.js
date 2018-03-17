import React, { Component } from 'react';
import { withRouter, Link } from 'react-router-dom';
import Select from 'react-select';
import { ClipLoader } from 'react-spinners';

import AuthService from 'services/AuthService';

import logoPrevent from 'images/logo-prevent.svg';
import loginStripes from 'images/login-stripes.svg';

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_AUTH_PROVIDERS':
            return { authProviders: nextState.authProviders };
        case 'UPDATE_SELECTED_AUTH_PROVIDER':
            return { selectedAuthProvider: nextState.selectedAuthProvider };
        default:
            return prevState;
    }
};

class LoginPage extends Component {
    constructor(props) {
        super(props);

        this.state = {
            selectedAuthProvider: '',
            authProviders: null
        };
    }

    componentDidMount() {
        this.getAuthProviders();
    }

    getAuthProviders = () => {
        AuthService.updateAuthProviders()
            .then(() => {
                const authProviders = AuthService.getAuthProviders();
                this.update('UPDATE_AUTH_PROVIDERS', { authProviders });
                if (authProviders.length)
                    this.update('UPDATE_SELECTED_AUTH_PROVIDER', {
                        selectedAuthProvider: authProviders[0].id
                    });
            })
            .catch(error => {
                console.error(error);
            });
    };

    login = () => {
        const { selectedAuthProvider } = this.state;
        const authProvider = this.state.authProviders.find(obj => obj.id === selectedAuthProvider);
        if (authProvider) window.location = authProvider.loginUrl;
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    renderAuthProviders = () => {
        if (
            !this.state.authProviders ||
            AuthService.getAccessToken() ||
            !AuthService.getAuthProviders().length
        ) {
            return '';
        }
        const { selectedAuthProvider } = this.state;
        const options = this.state.authProviders
            .filter(obj => obj.enabled)
            .map(authProvider => ({ label: authProvider.name, value: authProvider.id }));
        const value = selectedAuthProvider || selectedAuthProvider.value;
        const handleChange = () => option => {
            this.update('UPDATE_SELECTED_AUTH_PROVIDER', { selectedAuthProvider: option });
        };
        return (
            <div className="py-8 items-center w-2/3">
                <div className="text-primary-600 pb-3">Select an auth provider</div>
                <Select
                    className="text-base-600 font-400 w-full"
                    value={value}
                    name="select-auth-providers"
                    simpleValue
                    clearable={false}
                    disabled={this.state.authProviders.length === 1}
                    onChange={handleChange()}
                    options={options}
                />
            </div>
        );
    };

    renderLoginButton = () => {
        if (!this.state.authProviders) {
            return (
                <div className="border-t border-base-300 p-6 w-full text-center">
                    <button className="p-3 px-6 rounded-sm bg-primary-600 text-white uppercase text-center tracking-wide">
                        <ClipLoader color="white" loading size={15} />
                    </button>
                </div>
            );
        }
        if (AuthService.getAccessToken() || !AuthService.getAuthProviders().length) {
            return (
                <div className="border-t border-base-300 p-8 w-full text-center">
                    <Link
                        className="p-3 px-6 rounded-sm bg-primary-600 text-white uppercase text-center tracking-wide no-underline"
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
                    className="p-3 px-6 rounded-sm bg-primary-600 text-white uppercase text-center tracking-wide"
                    onClick={this.login}
                >
                    Login
                </button>
            </div>
        );
    };

    render() {
        return (
            <section className="flex flex-col items-center justify-center h-full bg-primary-600">
                <div className="flex flex-col items-center justify-center bg-white w-2/5 relative">
                    <img className="absolute pin-l pin-t" src={loginStripes} alt="" />
                    <img
                        className="absolute pin-r pin-b transform-rotate-half-turn"
                        src={loginStripes}
                        alt=""
                    />
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

export default withRouter(LoginPage);
