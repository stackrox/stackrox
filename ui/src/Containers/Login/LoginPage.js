import React, { Component } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import Select from 'react-select';
import { ClipLoader } from 'react-spinners';

import Logo from 'Components/icons/logo';
import AuthService from 'Providers/AuthService';

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
    static propTypes = {
        history: ReactRouterPropTypes.history.isRequired
    };

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

    renderLogin = () => {
        if (!this.state.authProviders) {
            return (
                <button className="p-4 h-12 text-lg rounded-sm bg-white text-transparent uppercase text-center">
                    <ClipLoader color="black" loading size={15} />
                </button>
            );
        }
        if (AuthService.getAccessToken() || !AuthService.getAuthProviders().length) {
            const buttonHandler = () => () => {
                this.props.history.go('/');
            };
            return (
                <button
                    className="p-4 h-12 text-lg rounded-sm bg-white text-transparent uppercase text-center mt-8"
                    onClick={buttonHandler()}
                >
                    Go to Dashboard
                </button>
            );
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
            <div className="flex flex-row mt-8">
                <Select
                    className="text-base-600 font-400 w-64"
                    value={value}
                    name="select-auth-providers"
                    simpleValue
                    clearable={false}
                    disabled={this.state.authProviders.length === 1}
                    onChange={handleChange()}
                    options={options}
                />
                <button
                    className="ml-4 p-2 px-4 rounded-sm bg-white text-transparent uppercase text-center"
                    onClick={this.login}
                >
                    Login
                </button>
            </div>
        );
    };

    render() {
        return (
            <section className="flex flex-col items-center justify-center h-full login-bg">
                <div className="flex flex-col items-center justify-center mb-8">
                    <Logo className="fill-current text-white h-24 w-24" />
                    <h1 className="text-2xl font-sans text-white tracking-wide mb-4">
                        Welcome to StackRox Mitigate
                    </h1>
                    <h2 className="text-xl font-sans text-white tracking-wide text-center">
                        Addressing pre-runtime security use cases for your container deployments
                    </h2>
                    {this.renderLogin()}
                </div>
            </section>
        );
    }
}

export default withRouter(LoginPage);
