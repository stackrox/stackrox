import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import Raven from 'raven-js';
import * as Icons from 'react-feather';

class ErrorBoundary extends React.Component {
    static propTypes = {
        children: PropTypes.node.isRequired
    };

    constructor(props) {
        super(props);
        this.state = { hasError: false };
    }

    static getDerivedStateFromError() {
        // Update state so the next render will show the fallback UI.
        return { hasError: true };
    }

    componentDidCatch(error, errorInfo) {
        // You can also log the error to an error reporting service
        Raven.captureException(error, { extra: errorInfo });
    }

    render() {
        if (this.state.hasError) {
            // You can render any custom fallback UI
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
