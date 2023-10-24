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
import { configure as mobxConfigure } from 'mobx';
import { setDiagnosticsOptions } from 'monaco-yaml';

// Advanced Cluster Security extensions to PatternFly styles
import 'css/acs.css';
// We need the following file, to smooth out rough edges, as we migrate to PatternFly
import 'css/trumps.css';

import ErrorBoundary from 'Components/PatternFly/ErrorBoundary/ErrorBoundary';
import AppPage from 'Containers/AppPage';
import { ThemeProvider } from 'Containers/ThemeProvider';
import configureStore from 'store/configureStore';
import installRaven from 'installRaven';
import { ThunkAction, ThunkDispatch } from 'redux-thunk';
import { fetchFeatureFlagsThunk } from './reducers/featureFlags';
import { fetchPublicConfigThunk } from './reducers/publicConfig';
import { fetchCentralCapabilitiesThunk } from './reducers/centralCapabilities';
import configureApollo from './configureApolloClient';

// This enables syntax highlighting for the patternfly code editor
// Reference: https://github.com/patternfly/patternfly-react/tree/main/packages/react-code-editor#enable-yaml-syntax-highlighting
setDiagnosticsOptions({
    enableSchemaRequest: true,
    hover: true,
    completion: true,
    validate: true,
    format: true,
    schemas: [],
});

// We need to call this MobX utility function, to prevent the error
//   Uncaught Error: [MobX] There are multiple, different versions of MobX active. Make sure MobX is loaded only once or use `configure({ isolateGlobalState: true })`
// which occurs because both the PatternFly react-topology component and the Redoc API viewer library
// both load their own versions of MobX
mobxConfigure({ isolateGlobalState: true });

installRaven();

const rootNode = document.getElementById('root');
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
