import React, { useEffect, useState } from 'react';
import { ApolloProvider } from '@apollo/client';

import axios from 'services/instance';
import { addAuthInterceptors, getAccessToken } from 'services/AuthService';
import configureApolloClient from 'configureApolloClient';

import '@patternfly/react-styles/css/utilities/Accessibility/accessibility.css';
import '@patternfly/react-styles/css/utilities/Alignment/alignment.css';
import '@patternfly/react-styles/css/utilities/BackgroundColor/BackgroundColor.css';
import '@patternfly/react-styles/css/utilities/BoxShadow/box-shadow';
import '@patternfly/react-styles/css/utilities/Display/display.css';
import '@patternfly/react-styles/css/utilities/Flex/flex.css';
import '@patternfly/react-styles/css/utilities/Sizing/sizing.css';
import '@patternfly/react-styles/css/utilities/Spacing/spacing.css';
import '@patternfly/react-styles/css/utilities/Text/text.css';
import 'css/acs.css';

import { PageSection } from '@patternfly/react-core';
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

    return (
        <PageSection
            padding={{ default: 'noPadding' }}
            style={{ backgroundColor: 'var(--pf-global--BackgroundColor--light-300)' }}
        >
            {isAuth ? (
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
            )}
        </PageSection>
    );
}
