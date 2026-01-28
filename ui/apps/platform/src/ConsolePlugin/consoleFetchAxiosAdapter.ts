import { ApolloError } from '@apollo/client';
import { consoleFetch } from '@openshift-console/dynamic-plugin-sdk';
import { AxiosError, AxiosHeaders } from 'axios';
import type { InternalAxiosRequestConfig } from 'axios';

import type { AuthScope } from './ScopeContext';
import { ALL_NAMESPACES_KEY } from './constants';

export default function consoleFetchAxiosAdapter(
    baseUrl: string,
    config: InternalAxiosRequestConfig,
    getScope: () => AuthScope = () => ({})
) {
    const updatedHeaders = new AxiosHeaders(config.headers);
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
        .then(async (response: Response) => {
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
        .catch(async (error: unknown) => {
            if (
                typeof error === 'object' &&
                error !== null &&
                'response' in error &&
                error.response instanceof Response
            ) {
                // If the error contains response information for an HTTP 4xx or 5xx error, we can extract the message from the response body
                const { status, statusText } = error.response;
                const text = await error.response.text();
                const headers = new AxiosHeaders();
                const axiosResponse = { status, statusText, headers, config, data: text };
                // Preserve original error context by passing the original error object and response information
                throw new AxiosError(text, `${status}`, { headers }, undefined, axiosResponse);
            }

            if (error instanceof Error) {
                throw error;
            }

            throw new Error(String(error));
        });
}
