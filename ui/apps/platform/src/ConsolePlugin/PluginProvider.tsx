import { useMemo } from 'react';
import type { ReactNode } from 'react';
import { ApolloProvider } from '@apollo/client';

import axios from 'services/instance';
import configureApolloClient from 'init/configureApolloClient';
import { UserPermissionProvider } from 'providers/UserPermissionProvider';
import { FeatureFlagsProvider } from 'providers/FeatureFlagProvider';
import { MetadataProvider } from 'providers/MetadataProvider';
import { PublicConfigProvider } from 'providers/PublicConfigProvider';

import consoleFetchAxiosAdapter from './consoleFetchAxiosAdapter';
import PluginContent from './PluginContent';
import { ScopeProvider, useScope } from './ScopeContext';

const proxyBasePath = process.env.ACS_PROXY_BASE_PATH ?? '/proxy/central';
// Remove trailing slash if present, and _do not_ add it back when constructing the proxy URL on the client.
const proxyBaseURL =
    `/api/proxy/plugin/advanced-cluster-security/api-service${proxyBasePath}`.replace(/\/$/, '');
const apolloClient = configureApolloClient();

function PluginProviderContent({ children }: { children: ReactNode }) {
    const scopeRef = useScope();

    // The console requires a custom fetch implementation via `consoleFetch` to correctly pass headers such
    // as X-CSRFToken to API requests. All of our current code uses `axios` to make API requests, so we need
    // to override the default adapter to use `consoleFetch` instead of XMLHttpRequest.
    //
    // Setup axios adapter to read scopeRef.current at request time
    axios.defaults.adapter = (config) =>
        consoleFetchAxiosAdapter(proxyBaseURL, config, () => scopeRef.current);

    return (
        <ApolloProvider client={apolloClient}>
            <UserPermissionProvider>
                <FeatureFlagsProvider>
                    <MetadataProvider>
                        <PublicConfigProvider>
                            <PluginContent>{children}</PluginContent>
                        </PublicConfigProvider>
                    </MetadataProvider>
                </FeatureFlagsProvider>
            </UserPermissionProvider>
        </ApolloProvider>
    );
}

export function PluginProvider({ children }: { children: ReactNode }) {
    return (
        <ScopeProvider>
            <PluginProviderContent>{children}</PluginProviderContent>
        </ScopeProvider>
    );
}

// If there is any data that needs to be shared across plugin entry points that isn't covered by
// a general purpose hook, we can add it here.
export function usePluginContext() {
    return useMemo(() => ({}), []);
}
