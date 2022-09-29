/**
 * This file is intentionally `.tsx` so CRA will detect that the app can be compiled with TypeScript.
 * The rest of the files can be either TypeScript (.ts or .tsx) or JavaScript (.js).
 */

import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux';
import { Store } from 'redux';
import { ConnectedRouter } from 'connected-react-router';
import { createBrowserHistory as createHistory } from 'history';
import { ApolloProvider } from '@apollo/client';
import 'typeface-open-sans';
import 'typeface-open-sans-condensed';
import 'react-toastify/dist/ReactToastify.css';
import 'app.tw.css'; // this file is the main Tailwind entrypoint handled by react-scripts
import '@patternfly/react-core/dist/styles/base.css';
import '@patternfly/react-styles/css/utilities/Accessibility/accessibility.css';
import '@patternfly/react-styles/css/utilities/Alignment/alignment.css';
import '@patternfly/react-styles/css/utilities/BackgroundColor/BackgroundColor.css';
import '@patternfly/react-styles/css/utilities/BoxShadow/box-shadow';
import '@patternfly/react-styles/css/utilities/Display/display.css';
import '@patternfly/react-styles/css/utilities/Flex/flex.css';
import '@patternfly/react-styles/css/utilities/Sizing/sizing.css';
import '@patternfly/react-styles/css/utilities/Spacing/spacing.css';
import '@patternfly/react-styles/css/utilities/Text/text.css';

// Advanced Cluster Security extensions to PatternFly styles
import 'css/acs.css';
// We need the following file, to smooth out rough edges, as we migrate to PatternFly
import 'css/trumps.css';

import ErrorBoundary from 'Containers/ErrorBoundary';
import AppPage from 'Containers/AppPage';
import { ThemeProvider } from 'Containers/ThemeProvider';
import configureStore from 'store/configureStore';
import installRaven from 'installRaven';
import configureApollo from './configureApolloClient';

installRaven();

const rootNode = document.getElementById('root');
const history = createHistory();
const store = configureStore(undefined, history) as Store;
const apolloClient = configureApollo();

ReactDOM.render(
    <Provider store={store}>
        <ApolloProvider client={apolloClient}>
            <ConnectedRouter history={history}>
                <ThemeProvider>
                    <ErrorBoundary>
                        <AppPage />
                    </ErrorBoundary>
                </ThemeProvider>
            </ConnectedRouter>
        </ApolloProvider>
    </Provider>,
    rootNode
);
