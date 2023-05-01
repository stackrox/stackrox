import { combineReducers } from 'redux';
import { reducer as formReducer } from 'redux-form';
import { connectRouter } from 'connected-react-router';

import bindSelectors from 'utils/bindSelectors';
import apiTokens, { selectors as apiTokenSelectors } from './apitokens';
import auth, { selectors as authSelectors } from './auth';
import clusterInitBundles, { selectors as clusterInitBundleSelectors } from './clusterInitBundles';
import clusters, { selectors as clusterSelectors } from './clusters';
import feedback, { selectors as feedbackSelectors } from './feedback';
import formMessages, { selectors as formMessageSelectors } from './formMessages';
import integrations, { selectors as integrationSelectors } from './integrations';
import notifications, { selectors as notificationSelectors } from './notifications';
import featureFlags, { selectors as featureFlagSelectors } from './featureFlags';
import policies, { selectors as policySelectors } from './policies/reducer';
import roles, { selectors as roleSelectors } from './roles';
import searchAutoComplete, { selectors as searchAutoCompleteSelectors } from './searchAutocomplete';
import serverResponseStatus, {
    selectors as serverResponseStatusSelectors,
} from './serverResponseStatus';
import metadata, { selectors as metadataSelectors } from './metadata';
import loading, { selectors as loadingSelectors } from './loading';
import { selectors as routeSelectors } from './routes';
import network, { selectors as networkSelectors } from './network/reducer';
import groups, { selectors as groupsSelectors } from './groups';
import publicConfig, { selectors as publicConfigSelectors } from './publicConfig';
import telemetryConfig, { selectors as telemetryConfigSelectors } from './telemetryConfig';

// Reducers

const appReducer = combineReducers({
    apiTokens,
    auth,
    clusterInitBundles,
    clusters,
    feedback,
    formMessages,
    integrations,
    notifications,
    featureFlags,
    policies,
    roles,
    searchAutoComplete,
    serverResponseStatus,
    loading,
    metadata,
    network,
    groups,
    publicConfig,
    telemetryConfig,
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
const getFeedback = (state) => getApp(state).feedback;
const getFormMessages = (state) => getApp(state).formMessages;
const getIntegrations = (state) => getApp(state).integrations;
const getNotifications = (state) => getApp(state).notifications;
const getFeatureFlags = (state) => getApp(state).featureFlags;
const getPolicies = (state) => getApp(state).policies;
const getRoles = (state) => getApp(state).roles;
const getSearchAutocomplete = (state) => getApp(state).searchAutoComplete;
const getServerResponseStatus = (state) => getApp(state).serverResponseStatus;
const getLoadingStatus = (state) => getApp(state).loading;
const getMetadata = (state) => getApp(state).metadata;
const getNetwork = (state) => getApp(state).network;
const getRuleGroups = (state) => getApp(state).groups;
const getPublicConfig = (state) => getApp(state).publicConfig;
const getTelemetryConfig = (state) => getApp(state).telemetryConfig;

const boundSelectors = {
    ...bindSelectors(getAPITokens, apiTokenSelectors),
    ...bindSelectors(getAuth, authSelectors),
    ...bindSelectors(getClusterInitBundles, clusterInitBundleSelectors),
    ...bindSelectors(getClusters, clusterSelectors),
    ...bindSelectors(getFeedback, feedbackSelectors),
    ...bindSelectors(getFormMessages, formMessageSelectors),
    ...bindSelectors(getIntegrations, integrationSelectors),
    ...bindSelectors(getNotifications, notificationSelectors),
    ...bindSelectors(getFeatureFlags, featureFlagSelectors),
    ...bindSelectors(getPolicies, policySelectors),
    ...bindSelectors(getRoles, roleSelectors),
    ...bindSelectors(getRoute, routeSelectors),
    ...bindSelectors(getSearchAutocomplete, searchAutoCompleteSelectors),
    ...bindSelectors(getServerResponseStatus, serverResponseStatusSelectors),
    ...bindSelectors(getLoadingStatus, loadingSelectors),
    ...bindSelectors(getMetadata, metadataSelectors),
    ...bindSelectors(getNetwork, networkSelectors),
    ...bindSelectors(getRuleGroups, groupsSelectors),
    ...bindSelectors(getPublicConfig, publicConfigSelectors),
    ...bindSelectors(getTelemetryConfig, telemetryConfigSelectors),
};

export const selectors = {
    ...boundSelectors,
};
