import { ApolloClient } from 'apollo-client';
import { createHttpLink } from 'apollo-link-http';
import { InMemoryCache } from 'apollo-cache-inmemory';
import { setContext } from 'apollo-link-context';

import { getAccessToken } from 'services/AuthService';

const httpLink = createHttpLink({
    uri: '/api/graphql'
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
        cache: new InMemoryCache()
    });
}
