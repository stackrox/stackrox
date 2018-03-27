import { combineReducers } from 'redux';
import { reducer as formReducer } from 'redux-form';

import bindSelectors from 'utils/bindSelectors';
import alerts, { selectors as alertSelectors } from './alerts';
import authProviders, { selectors as authProviderSelectors } from './auth';
import benchmarks, { selectors as benchmarkSelectors } from './benchmarks';
import clusters, { selectors as clusterSelectors } from './clusters';
import deployments, { selectors as deploymentSelectors } from './risk';
import integrations, { selectors as integrationSelectors } from './integrations';
import policies, { selectors as policySelectors } from './policies';
import route, { selectors as routeSelectors } from './routes';

// Reducers

const appReducer = combineReducers({
    alerts,
    authProviders,
    benchmarks,
    clusters,
    deployments,
    integrations,
    policies
});

const rootReducer = combineReducers({
    route,
    form: formReducer,
    app: appReducer
});

export default rootReducer;

// Selectors

const getRoute = state => state.route;
const getApp = state => state.app;
const getAlerts = state => getApp(state).alerts;
const getAuthProviders = state => getApp(state).authProviders;
const getBenchmarks = state => getApp(state).benchmarks;
const getClusters = state => getApp(state).clusters;
const getDeployments = state => getApp(state).deployments;
const getIntegrations = state => getApp(state).integrations;
const getPolicies = state => getApp(state).policies;

const boundSelectors = {
    ...bindSelectors(getAlerts, alertSelectors),
    ...bindSelectors(getAuthProviders, authProviderSelectors),
    ...bindSelectors(getBenchmarks, benchmarkSelectors),
    ...bindSelectors(getClusters, clusterSelectors),
    ...bindSelectors(getDeployments, deploymentSelectors),
    ...bindSelectors(getIntegrations, integrationSelectors),
    ...bindSelectors(getPolicies, policySelectors),
    ...bindSelectors(getRoute, routeSelectors)
};

export const selectors = {
    ...boundSelectors
};
