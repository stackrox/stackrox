import { combineReducers } from 'redux';
import { reducer as formReducer } from 'redux-form';
import { connectRouter } from 'connected-react-router';

import bindSelectors from 'utils/bindSelectors';
import apiTokens, { selectors as apiTokenSelectors } from './apitokens';
import auth, { selectors as authSelectors } from './auth';
import clusterInitBundles, { selectors as clusterInitBundleSelectors } from './clusterInitBundles';
import clusters, { selectors as clusterSelectors } from './clusters';
import formMessages, { selectors as formMessageSelectors } from './formMessages';
import integrations, { selectors as integrationSelectors } from './integrations';
import notifications, { selectors as notificationSelectors } from './notifications';
import featureFlags, { selectors as featureFlagSelectors } from './featureFlags';
import globalSearch, { selectors as globalSearchSelectors } from './globalSearch';
import policies, { selectors as policySelectors } from './policies/reducer';
import roles, { selectors as roleSelectors } from './roles';
import searchAutoComplete, { selectors as searchAutoCompleteSelectors } from './searchAutocomplete';
import serverError, { selectors as serverErrorSelectors } from './serverError';
import secrets, { selectors as secretSelectors } from './secrets';
import metadata, { selectors as metadataSelectors } from './metadata';
import loading, { selectors as loadingSelectors } from './loading';
import { selectors as routeSelectors } from './routes';
import network, { selectors as networkSelectors } from './network/reducer';
import processes, { selectors as processSelectors } from './processes';
import groups, { selectors as groupsSelectors } from './groups';
import attributes, { selectors as attributesSelectors } from './attributes';
import pdfDownload, { selectors as pdfDownloadSelectors } from './pdfDownload';
import systemConfig, { selectors as systemConfigSelectors } from './systemConfig';

// Reducers

const appReducer = combineReducers({
    apiTokens,
    auth,
    clusterInitBundles,
    clusters,
    formMessages,
    integrations,
    notifications,
    featureFlags,
    globalSearch,
    policies,
    roles,
    searchAutoComplete,
    serverError,
    secrets,
    loading,
    metadata,
    network,
    processes,
    groups,
    attributes,
    pdfDownload,
    systemConfig,
});

const createRootReducer = (history) => {
    return combineReducers({
        router: connectRouter(history),
        form: formReducer,
        app: appReducer,
    });
};

export default createRootReducer;

// Selectors

const getRoute = (state) => state.router;
const getApp = (state) => state.app;
const getAPITokens = (state) => getApp(state).apiTokens;
const getAuth = (state) => getApp(state).auth;
const getClusterInitBundles = (state) => getApp(state).clusterInitBundles;
const getClusters = (state) => getApp(state).clusters;
const getFormMessages = (state) => getApp(state).formMessages;
const getIntegrations = (state) => getApp(state).integrations;
const getNotifications = (state) => getApp(state).notifications;
const getFeatureFlags = (state) => getApp(state).featureFlags;
const getGlobalSearches = (state) => getApp(state).globalSearch;
const getPolicies = (state) => getApp(state).policies;
const getRoles = (state) => getApp(state).roles;
const getSearchAutocomplete = (state) => getApp(state).searchAutoComplete;
const getServerError = (state) => getApp(state).serverError;
const getSecrets = (state) => getApp(state).secrets;
const getLoadingStatus = (state) => getApp(state).loading;
const getMetadata = (state) => getApp(state).metadata;
const getNetwork = (state) => getApp(state).network;
const getProcesses = (state) => getApp(state).processes;
const getRuleGroups = (state) => getApp(state).groups;
const getAttributes = (state) => getApp(state).attributes;
const getPdfDownload = (state) => getApp(state).pdfDownload;
const getSystemConfig = (state) => getApp(state).systemConfig;

const boundSelectors = {
    ...bindSelectors(getAPITokens, apiTokenSelectors),
    ...bindSelectors(getAuth, authSelectors),
    ...bindSelectors(getClusterInitBundles, clusterInitBundleSelectors),
    ...bindSelectors(getClusters, clusterSelectors),
    ...bindSelectors(getFormMessages, formMessageSelectors),
    ...bindSelectors(getIntegrations, integrationSelectors),
    ...bindSelectors(getNotifications, notificationSelectors),
    ...bindSelectors(getFeatureFlags, featureFlagSelectors),
    ...bindSelectors(getGlobalSearches, globalSearchSelectors),
    ...bindSelectors(getPolicies, policySelectors),
    ...bindSelectors(getRoles, roleSelectors),
    ...bindSelectors(getRoute, routeSelectors),
    ...bindSelectors(getSearchAutocomplete, searchAutoCompleteSelectors),
    ...bindSelectors(getServerError, serverErrorSelectors),
    ...bindSelectors(getSecrets, secretSelectors),
    ...bindSelectors(getLoadingStatus, loadingSelectors),
    ...bindSelectors(getMetadata, metadataSelectors),
    ...bindSelectors(getNetwork, networkSelectors),
    ...bindSelectors(getProcesses, processSelectors),
    ...bindSelectors(getRuleGroups, groupsSelectors),
    ...bindSelectors(getAttributes, attributesSelectors),
    ...bindSelectors(getPdfDownload, pdfDownloadSelectors),
    ...bindSelectors(getSystemConfig, systemConfigSelectors),
};

export const selectors = {
    ...boundSelectors,
};
