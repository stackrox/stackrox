import { ApolloError } from '@apollo/client';
import { consoleFetch } from '@openshift-console/dynamic-plugin-sdk';
import { AxiosError, InternalAxiosRequestConfig } from 'axios';

export default function consoleFetchAxiosAdapter(
    baseUrl: string,
    config: InternalAxiosRequestConfig
) {
    const updatedHeaders = { ...config.headers };
    // Note - in production authorization is handled in-cluster by the console and will overwrite this header. When
    // running locally, we need to inject the token manually to allow API requests to the ACS API.
    if (process.env.NODE_ENV === 'development' && process.env.ACS_CONSOLE_DEV_TOKEN) {
        updatedHeaders.Authorization = `Bearer ${process.env.ACS_CONSOLE_DEV_TOKEN}`;
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
            throw new AxiosError(error.message);
        });
}
