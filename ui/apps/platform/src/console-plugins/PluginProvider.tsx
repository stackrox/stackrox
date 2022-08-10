import * as React from 'react';
import { ApolloProvider } from '@apollo/client';

import axios from 'services/instance';
import { addAuthInterceptors } from 'services/AuthService';
import configureApolloClient from 'configureApolloClient';

import 'index.css';
import 'css/acs.css';
import 'css/trumps.css';

const baseURL = 'https://central-stackrox.apps.ui-08-08-hack-a-thon-3.openshift.infra.rox.systems';

const apolloClient = configureApolloClient();

axios.interceptors.request.use((config) => {
    return { ...config, baseURL };
});

// TODO We need a way to get the JWT into localStorage `access_token` for authenticated requests
// eslint-disable-next-line no-console
addAuthInterceptors(console.error);

export default function PluginProvider({ children }) {
    return <ApolloProvider client={apolloClient}>{children}</ApolloProvider>;
}
