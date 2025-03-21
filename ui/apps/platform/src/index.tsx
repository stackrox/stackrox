/**
 * This file is intentionally `.tsx` so CRA will detect that the app can be compiled with TypeScript.
 * The rest of the files can be either TypeScript (.ts or .tsx) or JavaScript (.js).
 */

import React from 'react';
import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';
import { AnyAction, Store } from 'redux';
import { ConnectedRouter } from 'connected-react-router';
import { createBrowserHistory as createHistory } from 'history';
import { ApolloProvider } from '@apollo/client';

import 'css.imports';

import { configure as mobxConfigure } from 'mobx';
import * as monaco from 'monaco-editor';
import { loader } from '@monaco-editor/react';
import { configureMonacoYaml } from 'monaco-yaml';

import ErrorBoundary from 'Components/PatternFly/ErrorBoundary/ErrorBoundary';
import AppPage from 'Containers/AppPage';
import configureStore from 'store/configureStore';
import installRaven from 'installRaven';
import { ThunkAction, ThunkDispatch } from 'redux-thunk';
import { fetchFeatureFlagsThunk } from './reducers/featureFlags';
import { fetchPublicConfigThunk } from './reducers/publicConfig';
import { fetchCentralCapabilitiesThunk } from './reducers/centralCapabilities';
import configureApollo from './configureApolloClient';

// This enables syntax highlighting for the patternfly code editor
// Reference: https://github.com/patternfly/patternfly-react/tree/main/packages/react-code-editor#enable-yaml-syntax-highlighting
configureMonacoYaml(monaco, {
    enableSchemaRequest: true,
    hover: true,
    completion: true,
    validate: true,
    format: true,
    schemas: [],
});
loader.config({ monaco });

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
const store = configureStore(undefined, history) as Store;
const apolloClient = configureApollo();

const dispatch = (action) =>
    (store.dispatch as ThunkDispatch<unknown, unknown, AnyAction>)(
        action as ThunkAction<void, unknown, unknown, AnyAction>
    );

dispatch(fetchFeatureFlagsThunk());
dispatch(fetchPublicConfigThunk());
dispatch(fetchCentralCapabilitiesThunk());

root.render(
    <Provider store={store}>
        <ApolloProvider client={apolloClient}>
            {/*
             (dv 2024-05-01)
             ConnectedRouter does not explicitly declare `children` as a prop, which is expected by React types >=18
             so we need to use `@ts-expect-error` to suppress the type error
            */}
            {/* @ts-expect-error `connected-react-router does not support React 18 */}
            <ConnectedRouter history={history}>
                <ErrorBoundary>
                    <AppPage />
                </ErrorBoundary>
            </ConnectedRouter>
        </ApolloProvider>
    </Provider>
);
