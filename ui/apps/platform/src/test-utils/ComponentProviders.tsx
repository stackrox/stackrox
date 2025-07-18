import React from 'react';
import { Store, createStore } from 'redux';
import { Provider } from 'react-redux';
import { BrowserRouter as Router } from 'react-router-dom';
// import { HistoryRouter as Router } from 'redux-first-history/rr6';
import { CompatRouter } from 'react-router-dom-v5-compat';
import { ApolloProvider } from '@apollo/client';

import configureApolloClient from 'init/configureApolloClient';

export type ComponentTestProviderProps = {
    children: React.ReactNode;
    reduxStore?: Store;
};

const defaultStore = createStore(() => ({}), {});

// Provides the base application providers for testing
export default function ComponentTestProviders({
    children,
    reduxStore = defaultStore,
}: ComponentTestProviderProps) {
    return (
        <Provider store={reduxStore}>
            <Router>
                <CompatRouter>
                    <ApolloProvider client={configureApolloClient()}>{children}</ApolloProvider>
                </CompatRouter>
            </Router>
        </Provider>
    );
}
