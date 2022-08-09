import * as React from 'react';
import { ApolloClient, ApolloProvider, InMemoryCache } from '@apollo/client';

import ViolationsPage from 'Containers/Violations/ViolationsPage';
import axios from 'services/instance';
import { addAuthInterceptors } from 'services/AuthService';

const baseURL = 'https://central-stackrox.apps.ui-08-08-hack-a-thon-3.openshift.infra.rox.systems';

const apolloClient = new ApolloClient({
    uri: `${baseURL}/api/graphql`,
    cache: new InMemoryCache(),
});

axios.interceptors.request.use((config) => {
    return { ...config, baseURL };
});

// TODO We need a way to get the JWT into localStorage `access_token` for authenticated requests
addAuthInterceptors(console.error);

export default function Violations() {
    return (
        <ApolloProvider client={apolloClient}>
            <ViolationsPage />
        </ApolloProvider>
    );
}
