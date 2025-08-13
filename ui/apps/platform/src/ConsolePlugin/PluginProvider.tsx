import React, { useMemo, type ReactNode } from 'react';
import { ApolloProvider } from '@apollo/client';

import axios from 'services/instance';
import { UserPermissionProvider } from 'providers/UserPermissionProvider';
import { FeatureFlagsProvider } from 'providers/FeatureFlagProvider';

import configureApolloClient from '../init/configureApolloClient';
import consoleFetchAxiosAdapter from './consoleFetchAxiosAdapter';
import PluginContent from './PluginContent';

// The console requires a custom fetch implementation via `consoleFetch` to correctly pass headers such
// as X-CSRFToken to API requests. All of our current code uses `axios` to make API requests, so we need
// to override the default adapter to use `consoleFetch` instead of XMLHttpRequest.
const proxyBaseURL = '/api/proxy/plugin/advanced-cluster-security/api-service';
axios.defaults.adapter = (config) => consoleFetchAxiosAdapter(proxyBaseURL, config);

const apolloClient = configureApolloClient();

export function PluginProvider({ children }: { children: ReactNode }) {
    return (
        <ApolloProvider client={apolloClient}>
            <UserPermissionProvider>
                <FeatureFlagsProvider>
                    <PluginContent>{children}</PluginContent>
                </FeatureFlagsProvider>
            </UserPermissionProvider>
        </ApolloProvider>
    );
}

// If there is any data that needs to be shared across plugin entry points that isn't covered by
// a general purpose hook, we can add it here.
export function usePluginContext() {
    return useMemo(() => ({}), []);
}
