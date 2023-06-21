import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { ClipLoader } from 'react-spinners';
import { formValueSelector, propTypes as reduxFormPropTypes, reduxForm } from 'redux-form';
import { Alert, Button, Title, TitleSizes } from '@patternfly/react-core';

import { AUTH_STATUS } from 'reducers/auth';
import { selectors } from 'reducers';
import { ThemeContext } from 'Containers/ThemeProvider';
import LoadingSection from 'Components/PatternFly/LoadingSection';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxPasswordField from 'Components/forms/ReduxPasswordField';
import Labeled from 'Components/Labeled';
import CollapsibleAnimatedDiv from 'Components/animations/CollapsibleAnimatedDiv';
import BrandLogo from 'Components/PatternFly/BrandLogo';

import ServerStatusBanner from '../MainPage/Banners/ServerStatusBanner';
import LoginNotice from './LoginNotice';

import { loginWithBasicAuth } from '../../services/AuthService';
import { parseAndDecodeFragment } from '../../utils/parseAndDecodeFragment';

const unknownErrorResponse = {
    error: 'Unknown error',
};

const authProvidersToSelectOptions = (authProviders) =>
    authProviders.map((authProvider) => ({
        label: authProvider.name,
        value: authProvider.id,
    }));

class LoginPage extends Component {
    static propTypes = {
        authStatus: PropTypes.oneOf(Object.keys(AUTH_STATUS).map((key) => AUTH_STATUS[key]))
            .isRequired,
        authProviders: PropTypes.arrayOf(PropTypes.object).isRequired,
        authProviderResponse: PropTypes.shape({
            error: PropTypes.string,
            error_description: PropTypes.string,
            error_uri: PropTypes.string,
        }).isRequired,
        formValues: PropTypes.shape({
            authProvider: PropTypes.string,
            username: PropTypes.string,
            password: PropTypes.string,
        }).isRequired,
        authorizeRoxctlMode: PropTypes.bool,
        ...reduxFormPropTypes,
    };

    static contextType = ThemeContext;

    static defaultProps = {
        authorizeRoxctlMode: false,
    };

    constructor(props) {
        super(props);
        const { authProviderResponse } = props;
        this.state = {
            loggingIn: false,
            authProviderResponse,
        };
    }

    getSelectedAuthProvider(formValues) {
        const { authProviders } = this.props;
        return authProviders.find((ap) => ap.id === formValues.authProvider);
    }

    login = (formValues) => {
        const authProvider = this.getSelectedAuthProvider(formValues);
        if (!authProvider) {
            return;
        }
        const { authorizeRoxctlMode } = this.props;
        if (authorizeRoxctlMode) {
            if (authProvider.type === 'basic') {
                this.setState({
                    authProviderResponse: {
                        error: 'Cannot use username / password login to authorize roxctl',
                    },
                });
                return;
            }
            const parsedFragment = parseAndDecodeFragment(window.location);
            if (!parsedFragment.has('authorizeCallback')) {
                this.setState({
                    authProviderResponse: {
                        error: 'No authorize callback specified. Make sure you reach this page via the roxctl login command',
                    },
                });
                return;
            }
            this.setState({ loggingIn: true });
            window.location.assign(
                `${authProvider.loginUrl}?authorizeCallback=${parsedFragment.get(
                    'authorizeCallback'
                )}`
            );
            return;
        }
        if (authProvider.type === 'basic') {
            this.setState({ loggingIn: true });

            const { username, password } = formValues;
            loginWithBasicAuth(username, password, authProvider)
                .catch((e) => {
                    this.setState({
                        authProviderResponse: e?.response?.data || unknownErrorResponse,
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
        const { authProviderResponse } = this.state;

        if (authProviderResponse && authProviderResponse.error) {
            const errorKey = authProviderResponse.error.replace('_', ' ');
            const errorMsg = authProviderResponse.error_description || '';
            const errorLink = ((url) =>
                url ? (
                    <span>
                        (<a href={url}>more info</a>)
                    </span>
                ) : (
                    []
                ))(authProviderResponse.error_uri);
            return (
                <Alert variant="danger" isInline title={errorKey} className="pf-u-mb-md">
                    {errorMsg} {errorLink}
                </Alert>
            );
        }
        return null;
    };

    renderFields = () => {
        const { authStatus, authorizeRoxctlMode } = this.props;
        let { authProviders } = this.props;
        if (
            !authorizeRoxctlMode &&
            (authStatus === AUTH_STATUS.LOADING ||
                authStatus === AUTH_STATUS.LOGGED_IN ||
                authStatus === AUTH_STATUS.ANONYMOUS_ACCESS)
        ) {
            return null;
        }

        let title = 'Log in to your account';
        if (authorizeRoxctlMode) {
            authProviders = authProviders.filter((provider) => provider.type !== 'basic');
            title = 'Authorize roxctl';
            if (authProviders.length === 0) {
                return (
                    <Alert
                        variant="danger"
                        isInline
                        title="roxct-authorize-error"
                        className="pf-u-mb-md"
                    >
                        Only basic auth provider given. Authorizing roxctl only works with non-basic
                        auth provider. Configure an auth provider and try again.
                    </Alert>
                );
            }
        }

        const options = authProvidersToSelectOptions(authProviders);
        return (
            <div>
                <Title headingLevel="h2" size={TitleSizes['3xl']} className="pb-12">
                    {title}
                </Title>
                <Labeled label="Select an auth provider">
                    <ReduxSelectField
                        name="authProvider"
                        disabled={authProviders.length === 1}
                        onChange={this.onAuthProviderSelected}
                        options={options}
                    />
                </Labeled>
                <CollapsibleAnimatedDiv isOpen={this.isBasicAuthProviderSelected()}>
                    <Labeled label="Username">
                        <ReduxTextField name="username" />
                    </Labeled>
                    <Labeled label="Password">
                        <ReduxPasswordField name="password" />
                    </Labeled>
                </CollapsibleAnimatedDiv>
            </div>
        );
    };

    renderLoginButton = () => {
        const { authStatus, authorizeRoxctlMode } = this.props;
        if (authStatus === AUTH_STATUS.LOADING) {
            return (
                <div className="p-6 w-full text-center">
                    <button
                        type="button"
                        className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 text-center"
                    >
                        <ClipLoader color="white" loading size={15} />
                    </button>
                </div>
            );
        }
        if (
            !authorizeRoxctlMode &&
            (authStatus === AUTH_STATUS.LOGGED_IN || authStatus === AUTH_STATUS.ANONYMOUS_ACCESS)
        ) {
            return (
                <div className="p-8 w-full text-center">
                    <Link
                        className="p-3 px-6 rounded-sm bg-primary-600 hover:bg-primary-700 text-base-100 text-center no-underline"
                        to="/main/dashboard"
                    >
                        Go to Dashboard
                    </Link>
                </div>
            );
        }

        const {
            formValues: { username, password },
        } = this.props;
        const { loggingIn } = this.state;
        const isDisabled =
            loggingIn || (this.isBasicAuthProviderSelected() && (!username || !password));

        return (
            <Button
                type="submit"
                isDisabled={isDisabled}
                isBlock
                onClick={this.props.handleSubmit(this.login)}
            >
                {authorizeRoxctlMode ? 'Authorize' : 'Log in'}
            </Button>
        );
    };

    render() {
        return (
            <>
                <ServerStatusBanner />
                <main className="flex h-full items-center justify-center">
                    <div className="flex items-start">
                        <form
                            className="pf-u-background-color-100 w-128 theme-light"
                            onSubmit={this.props.handleSubmit(this.login)}
                        >
                            <div className="flex flex-col p-12 w-full">
                                {this.renderFields()}
                                <LoginNotice />
                                {this.renderAuthError()}
                                {this.renderLoginButton()}
                            </div>
                        </form>
                        <BrandLogo className="pf-u-p-2xl" />
                    </div>
                </main>
            </>
        );
    }
}

const loginFormId = 'login-form';

const selector = formValueSelector(loginFormId);

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getLoginAuthProviders,
    authStatus: selectors.getAuthStatus,
    authProviderResponse: selectors.getAuthProviderError,
    formValues: (state) => selector(state, 'authProvider', 'username', 'password'),
});

const Form = reduxForm({
    form: loginFormId,
})(connect(mapStateToProps, null)(LoginPage));

// the whole reason for this component to exist is to pass initial values to the form
// which are based on the Redux state. Yet because initialValues matter only when
// component is mounted, we cannot mount a component until we have everything to populate
// initial values (in this case the list of auth providers)
const LoadingOrForm = ({ authProviders, authorizeRoxctlMode = false }) => {
    if (!authProviders.length) {
        return <LoadingSection message="Loading..." />;
    }

    let availableAuthProviders = authProviders;
    if (authorizeRoxctlMode) {
        availableAuthProviders = authProviders.filter((provider) => provider.type !== 'basic');
    }

    const options = authProvidersToSelectOptions(availableAuthProviders);
    // In case of roxctl authorize mode, we filter out the basic auth provider. This could lead
    // to us having no auth provider within the initial values, hence we need to be able to handle
    // the empty list of auth providers here.
    const initialValues = { authProvider: options[0]?.value };
    return <Form initialValues={initialValues} authorizeRoxctlMode={authorizeRoxctlMode} />;
};

// yep, it's connect again, because we need to initialize form values from the state
export default connect(mapStateToProps, null)(LoadingOrForm);
