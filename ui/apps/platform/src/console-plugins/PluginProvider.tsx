import React, { useEffect, useState } from 'react';
import { ApolloProvider } from '@apollo/client';

import axios from 'services/instance';
import { addAuthInterceptors, getAccessToken } from 'services/AuthService';
import configureApolloClient from 'configureApolloClient';

import 'css/acs.css';

import PluginLogin from './PluginLogin';

let baseURL = localStorage.getItem('acs-base-url');

const apolloClient = configureApolloClient();

axios.interceptors.request.use((config) => {
    return { ...config, baseURL };
});

let userAuthenticated = !!getAccessToken() && !!baseURL;
// eslint-disable-next-line no-console
addAuthInterceptors(() => {
    userAuthenticated = !!getAccessToken() && !!baseURL;
});

export default function PluginProvider({ children }) {
    const [isAuth, setIsAuth] = useState(userAuthenticated);

    useEffect(() => {
        if (userAuthenticated !== isAuth) {
            setIsAuth(userAuthenticated);
        }
    });

    return isAuth ? (
        <ApolloProvider client={apolloClient}>{children}</ApolloProvider>
    ) : (
        <PluginLogin
            onLogin={(res) => {
                if (res.token) {
                    userAuthenticated = true;
                    setIsAuth(true);
                }
            }}
            onEndpointChange={(endpoint: string) => {
                baseURL = `https://${endpoint}`;
                localStorage.setItem('acs-base-url', baseURL);
            }}
        />
    );
}
