import React from 'react';
import { Store, createStore } from 'redux';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';
import { ApolloProvider } from '@apollo/client';

import configureApolloClient from 'init/configureApolloClient';

// TODO: rebase and revert this name change when we merge 16022
export type ComponentTestProvidersProps = {
    children: React.ReactNode;
    reduxStore?: Store;
};

const defaultStore = createStore(() => ({}), {});

// Provides the base application providers for testing
export default function ComponentTestProviders({
    children,
    reduxStore = defaultStore,
}: ComponentTestProvidersProps) {
    return (
        <Provider store={reduxStore}>
            <BrowserRouter>
                <ApolloProvider client={configureApolloClient()}>{children}</ApolloProvider>
            </BrowserRouter>
        </Provider>
    );
}
