import { ApolloClient, HttpLink, InMemoryCache, defaultDataIdFromObject } from '@apollo/client';
import { buildAxiosFetch } from '@lifeomic/axios-fetch';
import merge from 'lodash/merge';

import axios from 'services/instance';
import possibleTypes from './possibleTypes.json'; // see `scripts/generate-graphql-possible-types.js` file

const httpLink = new HttpLink({
    uri: '/api/graphql',
    // redirect requests through already configured Axios instance for:
    //  - consistency: auth logic (token header, redirects, retries with token refresh) works for GraphQL requests
    //  - testability: Cypress only supports XHR (not fetch), UI is more testable if we do everything with XHR
    fetch: buildAxiosFetch(axios, (config) => {
        // There is no requirement to pass operation name as a query from the API side.
        // The primary reasons for doing so:
        //   - dev-friendliness: easier to distinguish requests in browser dev tools
        //   - testability: easier to mock and wait for the request in e2e tests
        const { operationName } = JSON.parse(config.data);

        // set a long timeout for GraphQL requests, as an escape hatch for slow queries
        const modifiedConfig = { ...config, timeout: 60000 };

        return {
            ...modifiedConfig,
            url: `${config.url}?opname=${operationName}`,
        };
    }),
});

const defaultOptions = {
    watchQuery: {
        fetchPolicy: 'cache-and-network',
        nextFetchPolicy: 'cache-first',
    },
};

const cache = new InMemoryCache({
    possibleTypes,
    dataIdFromObject: (object) => {
        if (object.id && object.clusterID) {
            return `${object.clusterID}/${object.id}`;
        }
        return defaultDataIdFromObject(object);
    },
    typePolicies: {
        Image: {
            fields: {
                metadata: {
                    // Recursively merge image->metadata fields. This is required since the
                    // metadata object does not have a reliable unique ID and subsequent queries to
                    // the resolver will cause duplicate requests and lost cache data.
                    merge: (existing, incoming) => merge({}, existing, incoming),
                },
                name: {
                    // Name is an object without a unique ID, so we need to merge it manually.
                    merge: (existing, incoming) => merge({}, existing, incoming),
                },
            },
        },
        ImageCVECore: {
            keyFields: ['cve'],
        },
    },
});

export default function configureApolloClient() {
    return new ApolloClient({
        link: httpLink,
        defaultOptions,
        cache,
    });
}
