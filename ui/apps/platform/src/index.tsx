/**
 * This file is intentionally `.tsx` so CRA will detect that the app can be compiled with TypeScript.
 * The rest of the files can be either TypeScript (.ts or .tsx) or JavaScript (.js).
 */

import React from 'react';
import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';

// We needed to backpedal to the react-router-v5-compat layer in order to be compatible with the console plugin API.
// To reverse this change once the console plugin API is updated to support react-router-dom@v6 we need to:
// 1. Remove the react-router-dom-v5-compat dependency and the <CompatRouter> wrapper below
// 2. Remove the react-router-dom dependency and the <Router> wrapper below, uncommenting the redux-first-history/rr6 import
// 3. Replace imports from react-router-dom-v5-compat with react-router-dom throughout the codebase
// import { HistoryRouter as Router } from 'redux-first-history/rr6';
import { CompatRouter } from 'react-router-dom-v5-compat';
import { ConnectedRouter } from 'connected-react-router';
import { createBrowserHistory as createHistory } from 'history';

import { AnyAction } from 'redux';
import { ThunkAction, ThunkDispatch } from 'redux-thunk';
import { ApolloProvider } from '@apollo/client';

import 'css.imports';

import { configure as mobxConfigure } from 'mobx';

import ErrorBoundary from 'Components/PatternFly/ErrorBoundary/ErrorBoundary';
import AppPage from 'Containers/AppPage';
import configureStore from 'init/configureStore';
import installRaven from 'init/installRaven';
import configureApollo from 'init/configureApolloClient';
import { FeatureFlagsProvider } from 'providers/FeatureFlagProvider';
import { PublicConfigProvider } from 'providers/PublicConfigProvider';
import { TelemetryConfigProvider } from 'providers/TelemetryConfigProvider';
import { MetadataProvider } from 'providers/MetadataProvider';
import ReduxUserPermissionProvider from 'Containers/ReduxUserPermissionProvider';
import { fetchCentralCapabilitiesThunk } from './reducers/centralCapabilities';

// We need to call this MobX utility function, to prevent the error
//   Uncaught Error: [MobX] There are multiple, different versions of MobX active. Make sure MobX is loaded only once or use `configure({ isolateGlobalState: true })`
// which occurs because both the PatternFly react-topology component and the Redoc API viewer library
// both load their own versions of MobX
mobxConfigure({ isolateGlobalState: true });

installRaven();

const rootNode = document.getElementById('root');
/* @ts-expect-error `createRoot` expects a non-null argument */
const root = createRoot(rootNode);
const history = createHistory();
const store = configureStore(undefined, history);
const apolloClient = configureApollo();

const dispatch = (action) =>
    (store.dispatch as ThunkDispatch<unknown, unknown, AnyAction>)(
        action as ThunkAction<void, unknown, unknown, AnyAction>
    );

dispatch(fetchCentralCapabilitiesThunk());

root.render(
    <Provider store={store}>
        <FeatureFlagsProvider>
            <ReduxUserPermissionProvider>
                <PublicConfigProvider>
                    <TelemetryConfigProvider>
                        <MetadataProvider>
                            <ApolloProvider client={apolloClient}>
                                <ConnectedRouter history={history}>
                                    <CompatRouter>
                                        <ErrorBoundary>
                                            <AppPage />
                                        </ErrorBoundary>
                                    </CompatRouter>
                                </ConnectedRouter>
                            </ApolloProvider>
                        </MetadataProvider>
                    </TelemetryConfigProvider>
                </PublicConfigProvider>
            </ReduxUserPermissionProvider>
        </FeatureFlagsProvider>
    </Provider>
);
