import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import Raven from 'raven-js';
import * as Icons from 'react-feather';

class ErrorBoundary extends Component {
    static propTypes = {
        location: ReactRouterPropTypes.location.isRequired,
        children: PropTypes.node.isRequired
    };

    state = {
        showError: false
    };

    // TODO: replace with getDerivedStateFromProps after upgrading to React 16.3
    componentWillReceiveProps(nextProps) {
        if (this.props.location !== nextProps.location) {
            // navigation happened, invalidate state assuming new component tree may fix the problem
            this.setState({ showError: false });
        }
    }

    componentDidCatch(error, errorInfo) {
        this.setState({ showError: true });
        Raven.captureException(error, { extra: errorInfo });
    }

    render() {
        if (this.state.showError) {
            return (
                <div className="flex h-full items-center justify-center bg-base-100 text-base-600">
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
