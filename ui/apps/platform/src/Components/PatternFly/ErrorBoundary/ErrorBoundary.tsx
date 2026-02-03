import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { useLocation } from 'react-router-dom-v5-compat';
import Raven from 'raven-js';

import { getAnalytics } from 'init/initializeAnalytics';
import { PAGE_CRASH, getRedactedOriginProperties } from 'hooks/useAnalytics';
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

class ErrorBoundary extends Component<Props, State> {
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

        try {
            // Note - we are using the bare analytics object here instead of the useAnalytics hook
            // because we want to avoid moving dependent providers out of the ErrorBoundary component.
            const analytics = getAnalytics();
            if (analytics) {
                // Extract first component from component stack for context
                const componentStackLines = errorInfo.componentStack?.split('\n') ?? [];
                const firstComponent =
                    componentStackLines.find((line) => line.trim().startsWith('at '))?.trim() ??
                    'Unknown';

                analytics
                    .track(
                        PAGE_CRASH,
                        {
                            errorName: error.name || 'Error',
                            errorMessage: error.message.slice(0, 150), // Truncate to avoid sending too much data
                            componentStack: firstComponent,
                            pathname: this.props.location,
                        },
                        {
                            context: {
                                page: getRedactedOriginProperties(window.location.toString()),
                            },
                        }
                    )
                    .catch((analyticsError) => {
                        Raven.captureException(analyticsError);
                    });
            }
        } catch (analyticsError) {
            Raven.captureException(analyticsError);
        }
    }

    render() {
        if (this.state.hasError) {
            return <ErrorBoundaryPage error={this.state.error} errorInfo={this.state.errorInfo} />;
        }

        return this.props.children;
    }
}

function ErrorBoundaryWrapper({ children }: { children: ReactNode }) {
    const location = useLocation();
    return <ErrorBoundary location={location.pathname}>{children}</ErrorBoundary>;
}

// Encapsulate ErrorBoundaryWrapper as implementation detail,
// especially since ErrorBoundary appears in Find results.
// eslint-disable-next-line limited/react-export-default
export default ErrorBoundaryWrapper;
