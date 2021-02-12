import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import Raven from 'raven-js';
import { Copy, XSquare } from 'react-feather';
import { CopyToClipboard } from 'react-copy-to-clipboard';

function ErrorFieldDetails({ heading, message }) {
    return (
        <details className="mb-2">
            <summary>
                {heading}{' '}
                <CopyToClipboard text={message}>
                    <button
                        aria-label="Copy error message to clipboard"
                        type="button"
                        className="btn-tertiary pt-1 h-4 w-4"
                    >
                        <Copy className="h-4 w-4" />
                    </button>
                </CopyToClipboard>
            </summary>
            <pre className="bg-base-200 max-h-64 p-2 overflow-auto">{message}</pre>
        </details>
    );
}

class ErrorBoundary extends Component {
    static propTypes = {
        location: ReactRouterPropTypes.location.isRequired,
        children: PropTypes.node.isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            hasError: false,
            errorLocation: null,
            error: null,
            errorInfo: null,
        };
    }

    static getDerivedStateFromProps(nextProps, state) {
        if (state.hasError && nextProps.location !== state.errorLocation) {
            // stop showing error on location change to allow user to navigate after error happens
            return { hasError: false, errorLocation: null, error: null, errorInfo: null };
        }
        return null;
    }

    componentDidCatch(error, errorInfo) {
        this.setState({ hasError: true, errorLocation: this.props.location, error, errorInfo });
        // log error to the server
        Raven.captureException(error, { extra: errorInfo });
    }

    render() {
        if (this.state.hasError) {
            const { error, errorInfo } = this.state;
            return (
                <div
                    className="flex h-full items-center justify-center bg-base-100 text-base-600"
                    data-testid="error-boundary"
                >
                    <XSquare size="48" />
                    <div className="p-2 text-lg">
                        <div className="mb-2">
                            <p>
                                We&apos;re sorry — something&apos;s gone wrong. The error has been
                                logged.
                            </p>
                            <p>
                                Please try to refresh the page or navigate to some other part of the
                                application.
                            </p>
                        </div>
                        <div className="max-w-128 pl-2">
                            {!!error?.message && (
                                <ErrorFieldDetails
                                    heading="Error message"
                                    message={error.message}
                                />
                            )}
                            {!!error?.stack && (
                                <ErrorFieldDetails heading="Stack trace" message={error.stack} />
                            )}
                            {!!errorInfo?.componentStack && (
                                <ErrorFieldDetails
                                    heading="Component stack"
                                    message={errorInfo.componentStack}
                                />
                            )}
                        </div>
                    </div>
                </div>
            );
        }

        return this.props.children;
    }
}

export default withRouter(ErrorBoundary);
