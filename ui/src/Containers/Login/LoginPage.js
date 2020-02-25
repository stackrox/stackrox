import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { ClipLoader } from 'react-spinners';
import { reduxForm, formValueSelector, propTypes as reduxFormPropTypes } from 'redux-form';

import LoadingSection from 'Components/LoadingSection';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxPasswordField from 'Components/forms/ReduxPasswordField';
import UnreachableWarning from 'Containers/UnreachableWarning';

import logoPlatform from 'images/logo-platform.svg';

import { AUTH_STATUS } from 'reducers/auth';
import { selectors } from 'reducers';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';

import { ThemeContext } from 'Containers/ThemeProvider';

import Labeled from 'Components/Labeled';
import posed from 'react-pose';
import AppWrapper from '../AppWrapper';
import LoginNotice from './LoginNotice';
import { loginWithBasicAuth } from '../../services/AuthService';

const unknownErrorResponse = {
    error: 'Unknown error'
};

const CollapsibleContent = posed.div({
    closed: { height: 0 },
    open: { height: 'inherit' }
});

const authProvidersToSelectOptions = authProviders =>
    authProviders.map(authProvider => ({
        label: authProvider.name,
        value: authProvider.id
    }));

class LoginPage extends Component {
    static propTypes = {
        authStatus: PropTypes.oneOf(Object.keys(AUTH_STATUS).map(key => AUTH_STATUS[key]))
            .isRequired,
        authProviders: PropTypes.arrayOf(PropTypes.object).isRequired,
        authProviderResponse: PropTypes.shape({
            error: PropTypes.string,
            error_description: PropTypes.string,
            error_uri: PropTypes.string
        }).isRequired,
        formValues: PropTypes.shape({
            authProvider: PropTypes.string,
            username: PropTypes.string,
            password: PropTypes.string
        }).isRequired,
        ...reduxFormPropTypes
    };

    static contextType = ThemeContext;

    constructor(props) {
        super(props);
        const { authProviderResponse } = props;
        this.state = {
            loggingIn: false,
            authProviderResponse
        };
    }

    getSelectedAuthProvider(formValues) {
        const { authProviders } = this.props;
        return authProviders.find(ap => ap.id === formValues.authProvider);
    }

    login = formValues => {
        const authProvider = this.getSelectedAuthProvider(formValues);
        if (!authProvider) return;
        if (authProvider.type === 'basic') {
            this.setState({ loggingIn: true });

            const { username, password } = formValues;
            loginWithBasicAuth(username, password, authProvider)
                .catch(e => {
                    this.setState({
                        authProviderResponse: e?.response?.data || unknownErrorResponse
                    });
                })
                .finally(() => {
                    this.setState({ loggingIn: false });
                });
        } else {
            window.location = authProvider.loginUrl; // redirect to external URL, so no react-router
        }
    };

    dismissAuthError = () => this.setState({ authProviderResponse: null });

    onAuthProviderSelected = () => {
        const { change } = this.props;
        change('username', '');
        change('password', '');
    };

    isBasicAuthProviderSelected() {
        return this.getSelectedAuthProvider(this.props.formValues)?.type === 'basic';
    }

    renderAuthError = () => {
        const fg = 'alert-800';
        const bg = 'alert-200';
        const color = `text-${fg} bg-${bg}`;
        const { authProviderResponse } = this.state;
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
        if (authProviderResponse && authProviderResponse.error) {
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

    renderFields = () => {
        const { authStatus, authProviders } = this.props;
        if (
            authStatus === AUTH_STATUS.LOADING ||
            authStatus === AUTH_STATUS.LOGGED_IN ||
            authStatus === AUTH_STATUS.ANONYMOUS_ACCESS
        ) {
            return null;
        }

        const options = authProvidersToSelectOptions(authProviders);
        return (
            <div className="py-8 items-center w-2/3">
                <Labeled label={<p className="text-primary-700">Select an auth provider</p>}>
                    <ReduxSelectField
                        name="authProvider"
                        disabled={authProviders.length === 1}
                        onChange={this.onAuthProviderSelected}
                        options={options}
                    />
                </Labeled>
                <CollapsibleContent
                    className="overflow-hidden"
                    pose={this.isBasicAuthProviderSelected() ? 'open' : 'closed'}
                >
                    {this.renderUsernameAndPassword()}
                </CollapsibleContent>
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

        const {
            formValues: { username, password }
        } = this.props;
        const { loggingIn } = this.state;
        const disabled =
            loggingIn || (this.isBasicAuthProviderSelected() && (!username || !password));

        return (
            <div className="border-t border-base-300 p-6 w-full text-center">
                <button
                    type="button"
                    disabled={disabled}
                    className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 uppercase text-center tracking-wide"
                    onClick={this.props.handleSubmit(this.login)}
                >
                    Login
                </button>
            </div>
        );
    };

    renderUsernameAndPassword = () => {
        return (
            <>
                <Labeled label="Username">
                    <ReduxTextField name="username" />
                </Labeled>
                <Labeled label="Password">
                    <ReduxPasswordField name="password" />
                </Labeled>
            </>
        );
    };

    render() {
        const { isDarkMode } = this.context;

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
                    <form
                        className="flex flex-col items-center justify-center bg-base-100 w-2/5 md:w-3/5 xl:w-2/5 relative login-bg"
                        onSubmit={this.props.handleSubmit(this.login)}
                    >
                        <UnreachableWarning />
                        <div className="login-border-t h-1 w-full" />
                        <div className="flex flex-col items-center justify-center w-full">
                            <img className="h-40 h-40 py-6" src={logoPlatform} alt="StackRox" />
                            {this.renderFields()}
                        </div>
                        <LoginNotice />
                        {this.renderLoginButton()}
                    </form>
                </section>
            </AppWrapper>
        );
    }
}

const loginFormId = 'login-form';

const selector = formValueSelector(loginFormId);

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getLoginAuthProviders,
    authStatus: selectors.getAuthStatus,
    authProviderResponse: selectors.getAuthProviderError,
    formValues: state => selector(state, 'authProvider', 'username', 'password')
});

const Form = reduxForm({
    form: loginFormId
})(
    connect(
        mapStateToProps,
        null
    )(LoginPage)
);

// the whole reason for this component to exist is to pass initial values to the form
// which are based on the Redux state. Yet because initialValues matter only when
// component is mounted, we cannot mount a component until we have everything to populate
// initial values (in this case the list of auth providers)
const LoadingOrForm = ({ authProviders }) => {
    if (!authProviders.length) return <LoadingSection message="Loading..." />;

    const options = authProvidersToSelectOptions(authProviders);
    const initialValues = { authProvider: options[0].value };
    return <Form initialValues={initialValues} />;
};

// yep, it's connect again, because we need to initialize form values from the state
export default connect(
    mapStateToProps,
    null
)(LoadingOrForm);
