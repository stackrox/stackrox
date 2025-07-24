import React from 'react';
import { ApolloProvider } from '@apollo/client';

import axios from 'services/instance';
import configureApolloClient from '../init/configureApolloClient';
import consoleFetchAxiosAdapter from './consoleFetchAxiosAdapter';

// The console requires a custom fetch implementation via `consoleFetch` to correctly pass headers such
// as X-CSRFToken to API requests. All of our current code uses `axios` to make API requests, so we need
// to override the default adapter to use `consoleFetch` instead of XMLHttpRequest.
const proxyBaseURL = '/api/proxy/plugin/advanced-cluster-security/api-service';
axios.defaults.adapter = (config) => consoleFetchAxiosAdapter(proxyBaseURL, config);

const apolloClient = configureApolloClient();

export default function PluginProvider({ children }) {
    return <ApolloProvider client={apolloClient}>{children}</ApolloProvider>;
}
