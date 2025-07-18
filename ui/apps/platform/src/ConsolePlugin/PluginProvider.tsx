import React from 'react';
import { ApolloProvider } from '@apollo/client';

import axios from 'services/instance';
import configureApolloClient from '../init/configureApolloClient';

const baseURL = '/api/proxy/plugin/advanced-cluster-security/api-service/';

const apolloClient = configureApolloClient();

axios.interceptors.request.use((config) => {
    const updatedConfig = { ...config, baseURL };

    // Note - in production authorization is handled in-cluster by the console and will overwrite this header. When
    // running locally, we need to inject the token manually to allow API requests to the ACS API.
    if (process.env.NODE_ENV === 'development' && process.env.ACS_CONSOLE_DEV_TOKEN) {
        updatedConfig.headers.Authorization = `Bearer ${process.env.ACS_CONSOLE_DEV_TOKEN}`;
    }

    return updatedConfig;
});

export default function PluginProvider({ children }) {
    return <ApolloProvider client={apolloClient}>{children}</ApolloProvider>;
}
