import { combineReducers } from 'redux';
import { reducer as formReducer } from 'redux-form';

import bindSelectors from 'utils/bindSelectors';
import alerts, { selectors as alertSelectors } from './alerts';
import apiTokens, { selectors as apiTokenSelectors } from './apitokens';
import auth, { selectors as authSelectors } from './auth';
import clusters, { selectors as clusterSelectors } from './clusters';
import deployments, { selectors as deploymentSelectors } from './deployments';
import images, { selectors as imageSelectors } from './images';
import integrations, { selectors as integrationSelectors } from './integrations';
import notifications, { selectors as notificationSelectors } from './notifications';
import globalSearch, { selectors as globalSearchSelectors } from './globalSearch';
import policies, { selectors as policySelectors } from './policies/reducer';
import roles, { selectors as roleSelectors } from './roles';
import searchAutoComplete, { selectors as searchAutoCompleteSelectors } from './searchAutocomplete';
import serverError, { selectors as serverErrorSelectors } from './serverError';
import summaries, { selectors as summarySelectors } from './summaries';
import secrets, { selectors as secretSelectors } from './secrets';
import metadata, { selectors as metadataSelectors } from './metadata';
import dashboard, { selectors as dashboardSelectors } from './dashboard';
import loading, { selectors as loadingSelectors } from './loading';
import route, { selectors as routeSelectors } from './routes';
import network, { selectors as networkSelectors } from './network/reducer';
import processes, { selectors as processSelectors } from './processes';
import groups, { selectors as groupsSelectors } from './groups';
import attributes, { selectors as attributesSelectors } from './attributes';
import cli, { selectors as cliSelectors } from './cli';
import pdfDownload, { selectors as pdfDownloadSelectors } from './pdfDownload';
import license, { selectors as licenseSelectors } from './license';

// Reducers

const appReducer = combineReducers({
    alerts,
    apiTokens,
    auth,
    clusters,
    deployments,
    images,
    integrations,
    notifications,
    globalSearch,
    cli,
    policies,
    roles,
    searchAutoComplete,
    serverError,
    summaries,
    secrets,
    dashboard,
    loading,
    metadata,
    network,
    processes,
    groups,
    attributes,
    pdfDownload,
    license
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
const getClusters = state => getApp(state).clusters;
const getDeployments = state => getApp(state).deployments;
const getImages = state => getApp(state).images;
const getIntegrations = state => getApp(state).integrations;
const getNotifications = state => getApp(state).notifications;
const getGlobalSearches = state => getApp(state).globalSearch;
const getPolicies = state => getApp(state).policies;
const getRoles = state => getApp(state).roles;
const getSearchAutocomplete = state => getApp(state).searchAutoComplete;
const getServerError = state => getApp(state).serverError;
const getSummaries = state => getApp(state).summaries;
const getSecrets = state => getApp(state).secrets;
const getDashboard = state => getApp(state).dashboard;
const getLoadingStatus = state => getApp(state).loading;
const getMetadata = state => getApp(state).metadata;
const getNetwork = state => getApp(state).network;
const getProcesses = state => getApp(state).processes;
const getRuleGroups = state => getApp(state).groups;
const getAttributes = state => getApp(state).attributes;
const getCLI = state => getApp(state).cli;
const getPdfDownload = state => getApp(state).pdfDownload;
const getLicense = state => getApp(state).license;

const boundSelectors = {
    ...bindSelectors(getAlerts, alertSelectors),
    ...bindSelectors(getAPITokens, apiTokenSelectors),
    ...bindSelectors(getAuth, authSelectors),
    ...bindSelectors(getClusters, clusterSelectors),
    ...bindSelectors(getDeployments, deploymentSelectors),
    ...bindSelectors(getImages, imageSelectors),
    ...bindSelectors(getIntegrations, integrationSelectors),
    ...bindSelectors(getNotifications, notificationSelectors),
    ...bindSelectors(getGlobalSearches, globalSearchSelectors),
    ...bindSelectors(getPolicies, policySelectors),
    ...bindSelectors(getRoles, roleSelectors),
    ...bindSelectors(getRoute, routeSelectors),
    ...bindSelectors(getSearchAutocomplete, searchAutoCompleteSelectors),
    ...bindSelectors(getServerError, serverErrorSelectors),
    ...bindSelectors(getSummaries, summarySelectors),
    ...bindSelectors(getSecrets, secretSelectors),
    ...bindSelectors(getDashboard, dashboardSelectors),
    ...bindSelectors(getLoadingStatus, loadingSelectors),
    ...bindSelectors(getMetadata, metadataSelectors),
    ...bindSelectors(getNetwork, networkSelectors),
    ...bindSelectors(getProcesses, processSelectors),
    ...bindSelectors(getRuleGroups, groupsSelectors),
    ...bindSelectors(getAttributes, attributesSelectors),
    ...bindSelectors(getCLI, cliSelectors),
    ...bindSelectors(getPdfDownload, pdfDownloadSelectors),
    ...bindSelectors(getLicense, licenseSelectors)
};

export const selectors = {
    ...boundSelectors
};
