import { combineReducers } from 'redux';
import { reducer as formReducer } from 'redux-form';

import bindSelectors from 'utils/bindSelectors';
import alerts, { selectors as alertSelectors } from './alerts';
import apiTokens, { selectors as apiTokenSelectors } from './apitokens';
import auth, { selectors as authSelectors } from './auth';
import benchmarks, { selectors as benchmarkSelectors } from './benchmarks';
import clusters, { selectors as clusterSelectors } from './clusters';
import deployments, { selectors as deploymentSelectors } from './deployments';
import images, { selectors as imageSelectors } from './images';
import integrations, { selectors as integrationSelectors } from './integrations';
import notifications, { selectors as notificationSelectors } from './notifications';
import globalSearch, { selectors as globalSearchSelectors } from './globalSearch';
import policies, { selectors as policySelectors } from './policies/reducer';
import roles, { selectors as roleSelectors } from './roles';
import summaries, { selectors as summarySelectors } from './summaries';
import secrets, { selectors as secretSelectors } from './secrets';
import metadata, { selectors as metadataSelectors } from './metadata';
import dashboard, { selectors as dashboardSelectors } from './dashboard';
import loading, { selectors as loadingSelectors } from './loading';
import route, { selectors as routeSelectors } from './routes';
import network, { selectors as networkSelectors } from './network';
import processes, { selectors as processSelectors } from './processes';
import groups, { selectors as groupsSelectors } from './groups';
import attributes, { selectors as attributesSelectors } from './attributes';

// Reducers

const appReducer = combineReducers({
    alerts,
    apiTokens,
    auth,
    benchmarks,
    clusters,
    deployments,
    images,
    integrations,
    notifications,
    globalSearch,
    policies,
    roles,
    summaries,
    secrets,
    dashboard,
    loading,
    metadata,
    network,
    processes,
    groups,
    attributes
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
const getAPITokens = state => getApp(state).apiTokens;
const getAuth = state => getApp(state).auth;
const getBenchmarks = state => getApp(state).benchmarks;
const getClusters = state => getApp(state).clusters;
const getDeployments = state => getApp(state).deployments;
const getImages = state => getApp(state).images;
const getIntegrations = state => getApp(state).integrations;
const getNotifications = state => getApp(state).notifications;
const getGlobalSearches = state => getApp(state).globalSearch;
const getPolicies = state => getApp(state).policies;
const getRoles = state => getApp(state).roles;
const getSummaries = state => getApp(state).summaries;
const getSecrets = state => getApp(state).secrets;
const getDashboard = state => getApp(state).dashboard;
const getLoadingStatus = state => getApp(state).loading;
const getMetadata = state => getApp(state).metadata;
const getNetwork = state => getApp(state).network;
const getProcesses = state => getApp(state).processes;
const getRuleGroups = state => getApp(state).groups;
const getAttributes = state => getApp(state).attributes;

const boundSelectors = {
    ...bindSelectors(getAlerts, alertSelectors),
    ...bindSelectors(getAPITokens, apiTokenSelectors),
    ...bindSelectors(getAuth, authSelectors),
    ...bindSelectors(getBenchmarks, benchmarkSelectors),
    ...bindSelectors(getClusters, clusterSelectors),
    ...bindSelectors(getDeployments, deploymentSelectors),
    ...bindSelectors(getImages, imageSelectors),
    ...bindSelectors(getIntegrations, integrationSelectors),
    ...bindSelectors(getNotifications, notificationSelectors),
    ...bindSelectors(getGlobalSearches, globalSearchSelectors),
    ...bindSelectors(getPolicies, policySelectors),
    ...bindSelectors(getRoles, roleSelectors),
    ...bindSelectors(getRoute, routeSelectors),
    ...bindSelectors(getSummaries, summarySelectors),
    ...bindSelectors(getSecrets, secretSelectors),
    ...bindSelectors(getDashboard, dashboardSelectors),
    ...bindSelectors(getLoadingStatus, loadingSelectors),
    ...bindSelectors(getMetadata, metadataSelectors),
    ...bindSelectors(getNetwork, networkSelectors),
    ...bindSelectors(getProcesses, processSelectors),
    ...bindSelectors(getRuleGroups, groupsSelectors),
    ...bindSelectors(getAttributes, attributesSelectors)
};

export const selectors = {
    ...boundSelectors
};
