import React from 'react';
import { Store, createStore } from 'redux';
import { Provider } from 'react-redux';
import { Router } from 'react-router-dom';
import { ApolloProvider } from '@apollo/client';
import { createBrowserHistory } from 'history';

import configureApolloClient from 'configureApolloClient';

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
    const history = createBrowserHistory();

    return (
        <Provider store={reduxStore}>
            {/* <Router history={history}> */}
            <ApolloProvider client={configureApolloClient()}>{children}</ApolloProvider>
            {/* </Router> */}
        </Provider>
    );
}
