import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import Raven from 'raven-js';
import * as Icons from 'react-feather';

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
        };
    }

    static getDerivedStateFromProps(nextProps, state) {
        if (state.hasError && nextProps.location !== state.errorLocation) {
            // stop showing error on location change to allow user to navigate after error happens
            return { hasError: false, errorLocation: null };
        }
        return null;
    }

    componentDidCatch(error, errorInfo) {
        this.setState({ hasError: true, errorLocation: this.props.location });
        // log error to the server
        Raven.captureException(error, { extra: errorInfo });
    }

    render() {
        if (this.state.hasError) {
            return (
                <div
                    className="flex h-full items-center justify-center bg-base-100 text-base-600"
                    data-testid="error-boundary"
                >
                    <Icons.XSquare size="48" />
                    <div className="p-2 text-lg">
                        <p>
                            We&apos;re sorry — something&apos;s gone wrong. The error has been
                            logged.
                        </p>
                        <p>
                            Please try to refresh the page or navigate to some other part of the
                            application.
                        </p>
                    </div>
                </div>
            );
        }

        return this.props.children;
    }
}

export default withRouter(ErrorBoundary);
