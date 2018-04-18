import { combineReducers } from 'redux';
import { reducer as formReducer } from 'redux-form';

import bindSelectors from 'utils/bindSelectors';
import alerts, { selectors as alertSelectors } from './alerts';
import authProviders, { selectors as authProviderSelectors } from './auth';
import benchmarks, { selectors as benchmarkSelectors } from './benchmarks';
import clusters, { selectors as clusterSelectors } from './clusters';
import deployments, { selectors as deploymentSelectors } from './risk';
import images, { selectors as imageSelectors } from './images';
import integrations, { selectors as integrationSelectors } from './integrations';
import globalSearch, { selectors as globalSearchSelectors } from './globalSearch';
import policies, { selectors as policySelectors } from './policies';
import summaries, { selectors as summarySelectors } from './summaries';
import route, { selectors as routeSelectors } from './routes';

// Reducers

const appReducer = combineReducers({
    alerts,
    authProviders,
    benchmarks,
    clusters,
    deployments,
    images,
    integrations,
    globalSearch,
    policies,
    summaries
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
const getImages = state => getApp(state).images;
const getIntegrations = state => getApp(state).integrations;
const getGlobalSearches = state => getApp(state).globalSearch;
const getPolicies = state => getApp(state).policies;
const getSummaries = state => getApp(state).summaries;

const boundSelectors = {
    ...bindSelectors(getAlerts, alertSelectors),
    ...bindSelectors(getAuthProviders, authProviderSelectors),
    ...bindSelectors(getBenchmarks, benchmarkSelectors),
    ...bindSelectors(getClusters, clusterSelectors),
    ...bindSelectors(getDeployments, deploymentSelectors),
    ...bindSelectors(getImages, imageSelectors),
    ...bindSelectors(getIntegrations, integrationSelectors),
    ...bindSelectors(getGlobalSearches, globalSearchSelectors),
    ...bindSelectors(getPolicies, policySelectors),
    ...bindSelectors(getRoute, routeSelectors),
    ...bindSelectors(getSummaries, summarySelectors)
};

export const selectors = {
    ...boundSelectors
};
