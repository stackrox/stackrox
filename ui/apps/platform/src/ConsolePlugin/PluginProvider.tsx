import React, { useMemo } from 'react';
import { ApolloProvider } from '@apollo/client';

import axios from 'services/instance';
import configureApolloClient from '../init/configureApolloClient';
import consoleFetchAxiosAdapter from './consoleFetchAxiosAdapter';
import UserPermissionProvider from './UserPermissionProvider';
import PluginContent from './PluginContent';

// The console requires a custom fetch implementation via `consoleFetch` to correctly pass headers such
// as X-CSRFToken to API requests. All of our current code uses `axios` to make API requests, so we need
// to override the default adapter to use `consoleFetch` instead of XMLHttpRequest.
const proxyBaseURL = '/api/proxy/plugin/advanced-cluster-security/api-service';
axios.defaults.adapter = (config) => consoleFetchAxiosAdapter(proxyBaseURL, config);

const apolloClient = configureApolloClient();

export function PluginProvider({ children }: { children: React.ReactNode }) {
    return (
        <ApolloProvider client={apolloClient}>
            <UserPermissionProvider>
                <PluginContent>{children}</PluginContent>
            </UserPermissionProvider>
        </ApolloProvider>
    );
}

// If there is any data that needs to be shared across plugin entry points that isn't covered by
// a general purpose hook, we can add it here.
export function usePluginContext() {
    return useMemo(() => ({}), []);
}
