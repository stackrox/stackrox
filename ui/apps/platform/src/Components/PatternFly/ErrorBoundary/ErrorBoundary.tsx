import React, { Component, ErrorInfo, ReactNode } from 'react';
import { useLocation } from 'react-router-dom';
import Raven from 'raven-js';

import ErrorBoundaryPage from './ErrorBoundaryPage';

type Props = {
    location: string;
    children: ReactNode;
};

type State =
    | {
          hasError: false;
          errorLocation: null;
          error: null;
          errorInfo: null;
      }
    | {
          hasError: true;
          errorLocation: string;
          error: Error;
          errorInfo: ErrorInfo;
      };

class ErrorBoundaryClass extends Component<Props, State> {
    constructor(props: Props) {
        super(props);

        this.state = {
            hasError: false,
            errorLocation: null,
            error: null,
            errorInfo: null,
        };
    }

    static getDerivedStateFromProps(nextProps: Props, state: State) {
        if (state.hasError && nextProps.location !== state.errorLocation) {
            // stop showing error on location change to allow user to navigate after error happens
            return { hasError: false, errorLocation: null, error: null, errorInfo: null };
        }
        return null;
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        this.setState({ hasError: true, errorLocation: this.props.location, error, errorInfo });
        // log error to the server
        Raven.captureException(error, { extra: errorInfo });
    }

    render() {
        if (this.state.hasError) {
            return <ErrorBoundaryPage error={this.state.error} errorInfo={this.state.errorInfo} />;
        }

        return this.props.children;
    }
}

function ErrorBoundary({ children }: { children: ReactNode }) {
    const location = useLocation();
    return <ErrorBoundaryClass location={location.pathname}>{children}</ErrorBoundaryClass>;
}

export default ErrorBoundary;
