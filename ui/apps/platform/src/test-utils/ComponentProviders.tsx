import React from 'react';
import { Store, createStore } from 'redux';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';
import { ApolloProvider } from '@apollo/client';

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
    return (
        <Provider store={reduxStore}>
            <BrowserRouter>
                <ApolloProvider client={configureApolloClient()}>{children}</ApolloProvider>
            </BrowserRouter>
        </Provider>
    );
}
