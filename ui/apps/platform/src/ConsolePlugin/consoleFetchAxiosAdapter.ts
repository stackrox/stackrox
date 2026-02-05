import { ApolloError } from '@apollo/client';
import { consoleFetch } from '@openshift-console/dynamic-plugin-sdk';
import { AxiosError } from 'axios';
import type { InternalAxiosRequestConfig } from 'axios';

import type { AuthScope } from './ScopeContext';
import { ALL_NAMESPACES_KEY } from './constants';

export default function consoleFetchAxiosAdapter(
    baseUrl: string,
    config: InternalAxiosRequestConfig,
    getScope: () => AuthScope = () => ({})
) {
    const updatedHeaders = { ...config.headers };
    // Note - in production authorization is handled in-cluster by the console and will overwrite this header. When
    // running locally, we need to inject the token manually to allow API requests to the ACS API.
    if (process.env.NODE_ENV === 'development' && process.env.ACS_CONSOLE_DEV_TOKEN) {
        updatedHeaders.Authorization = `Bearer ${process.env.ACS_CONSOLE_DEV_TOKEN}`;
    }

    // Add scope headers to assist in authorization decisions
    const scope = getScope();
    const isScopedDataRequest =
        config.url?.includes('/api/graphql') || config.url?.includes('/v1/deployments');
    if (scope.namespace && isScopedDataRequest) {
        // A value of "all namespaces" is used by cluster admins to request data from all namespace
        // which we represent as a wildcard in the API.
        const acsAuthNamespaceScope =
            scope.namespace === ALL_NAMESPACES_KEY ? '*' : scope.namespace;
        updatedHeaders['ACS-AUTH-NAMESPACE-SCOPE'] = acsAuthNamespaceScope;
    }

    return consoleFetch(`${baseUrl}${config.url}`, {
        method: config.method?.toUpperCase() ?? 'GET',
        body: config.data,
        headers: updatedHeaders,
    })
        .then(async (response) => {
            const data = await response.text();

            // GraphQL request errors are JSON objects with an `errors` field an a HTTP status code of 200, so we
            // need to check for that and throw an ApolloError
            if (config.url?.startsWith('/api/graphql')) {
                const json = JSON.parse(data);
                if ('errors' in json) {
                    throw new ApolloError({ graphQLErrors: json.errors });
                }
            }

            return {
                ...response,
                config,
                data,
                // Converts `fetch` headers to an `axios` compatible headers object
                headers: Object.fromEntries(response.headers.entries()),
                request: response,
                statusText: response.statusText,
            };
        })
        .catch((error) => {
            // Preserve original error context by passing the original error object and stack trace
            const axiosError: AxiosError & { originalError?: Error } = new AxiosError(
                error.message,
                undefined,
                config,
                undefined,
                error.response
            );
            axiosError.stack = error.stack;
            // Attach the original error for further debugging if needed
            axiosError.originalError = error;
            throw axiosError;
        });
}
