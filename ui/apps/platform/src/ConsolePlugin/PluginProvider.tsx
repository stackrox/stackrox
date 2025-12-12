import { useEffect } from 'react';
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
import { ScopeProvider, useScopeContext } from './ScopeContext';

const proxyBaseURL = '/api/proxy/plugin/advanced-cluster-security/api-service';
const apolloClient = configureApolloClient();

// The console requires a custom fetch implementation via `consoleFetch` to correctly pass headers such
// as X-CSRFToken to API requests. All of our current code uses `axios` to make API requests, so we need
// to override the default adapter to use `consoleFetch` instead of XMLHttpRequest.
//
// This is first done at the module level to establish authentication headers for the initial request, and later
// may update when the scope context changes
axios.defaults.adapter = (config) => consoleFetchAxiosAdapter(proxyBaseURL, config);

function PluginProviderContent({ children }: { children: ReactNode }) {
    const { getScope } = useScopeContext();

    useEffect(() => {
        // When the scope changes, we need to update the axios adapter to use the new scope.
        axios.defaults.adapter = (config) =>
            consoleFetchAxiosAdapter(proxyBaseURL, config, getScope);
    }, [getScope]);

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
    return {};
}
