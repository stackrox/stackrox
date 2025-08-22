import React from 'react';
import type { ReactNode } from 'react';
import { createStore } from 'redux';
import type { Store } from 'redux';
import { Provider } from 'react-redux';
import { BrowserRouter as Router } from 'react-router-dom';
// import { HistoryRouter as Router } from 'redux-first-history/rr6';
import { CompatRouter } from 'react-router-dom-v5-compat';
import { ApolloProvider } from '@apollo/client';

import configureApolloClient from 'init/configureApolloClient';

export type ComponentTestProviderProps = {
    children: ReactNode;
    reduxStore?: Store;
};

const defaultStore = createStore(() => ({}), {});

// Provides the base application providers for testing
export default function ComponentTestProvider({
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
