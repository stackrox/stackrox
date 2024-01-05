import React from 'react';
import { Router } from 'react-router-dom';
import { ApolloProvider } from '@apollo/client';
import { createBrowserHistory } from 'history';

import configureApolloClient from 'configureApolloClient';

// Provides the base application providers for testing
export default function ComponentTestProviders({ children }: { children: React.ReactNode }) {
    const history = createBrowserHistory();

    return (
        <Router history={history}>
            <ApolloProvider client={configureApolloClient()}>{children}</ApolloProvider>
        </Router>
    );
}
