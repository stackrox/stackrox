import { ApolloClient } from 'apollo-client';
import { createHttpLink } from 'apollo-link-http';
import {
    InMemoryCache,
    IntrospectionFragmentMatcher,
    defaultDataIdFromObject
} from 'apollo-cache-inmemory';
import { buildAxiosFetch } from '@lifeomic/axios-fetch';

import axios from 'services/instance';
import introspectionQueryResultData from './fragmentTypes.json';

const httpLink = createHttpLink({
    uri: '/api/graphql',
    // redirect requests through already configured Axios instance for:
    //  - consistency: auth logic (token header, redirects, retries with token refresh) works for GraphQL requests
    //  - testability: Cypress only supports XHR (not fetch), UI is more testable if we do everything with XHR
    fetch: buildAxiosFetch(axios, config => {
        // There is no requirement to pass operation name as a query from the API side.
        // The primary reasons for doing so:
        //   - dev-friendliness: easier to distinguish requests in browser dev tools
        //   - testability: easier to mock and wait for the request in e2e tests
        const { operationName } = JSON.parse(config.data);
        return {
            ...config,
            url: `${config.url}?opname=${operationName}`
        };
    })
});

const fragmentMatcher = new IntrospectionFragmentMatcher({
    introspectionQueryResultData
});

const defaultOptions = {
    watchQuery: {
        fetchPolicy: 'cache-and-network'
    }
};

export default function() {
    return new ApolloClient({
        link: httpLink,
        defaultOptions,
        cache: new InMemoryCache({
            fragmentMatcher,
            dataIdFromObject: object => {
                if (object.id && object.clusterID) return `${object.clusterID}/${object.id}`;
                return defaultDataIdFromObject(object);
            }
        })
    });
}
