import { ApolloClient } from 'apollo-client';
import { createHttpLink } from 'apollo-link-http';
import {
    InMemoryCache,
    IntrospectionFragmentMatcher,
    defaultDataIdFromObject
} from 'apollo-cache-inmemory';
import { setContext } from 'apollo-link-context';
import { getAccessToken } from 'services/AuthService';
import introspectionQueryResultData from './fragmentTypes.json';

// I was able to get the browser into a state where graphql usage would fail with the error
// Network error: Failed to execute 'fetch' on 'Window': Request cannot be constructed from a URL that includes credentials: /api/graphql
// It turns out that axios always canonicalizes relative URLs but apollo doesn't.
// We never noticed this before because graphql was silently eating errors.
const uri = `${window.location.protocol}//${window.location.host}/api/graphql`;

const httpLink = createHttpLink({
    uri
});

const fragmentMatcher = new IntrospectionFragmentMatcher({
    introspectionQueryResultData
});

const authLink = setContext((_, { headers }) => {
    // get the authentication token from local storage if it exists
    const token = getAccessToken();
    // return the headers to the context so httpLink can read them
    return {
        headers: {
            ...headers,
            authorization: token ? `Bearer ${token}` : ''
        }
    };
});

export default function() {
    return new ApolloClient({
        link: authLink.concat(httpLink),
        cache: new InMemoryCache({
            fragmentMatcher,
            dataIdFromObject: object => {
                if (object.id && object.clusterID) return `${object.clusterID}/${object.id}`;
                return defaultDataIdFromObject(object);
            }
        })
    });
}
