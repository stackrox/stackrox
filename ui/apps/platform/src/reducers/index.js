import { combineReducers } from 'redux';
import { reducer as formReducer } from 'redux-form';
import { connectRouter } from 'connected-react-router';

import bindSelectors from 'utils/bindSelectors';
import apiTokens, { selectors as apiTokenSelectors } from './apitokens';
import auth, { selectors as authSelectors } from './auth';
import machineAccessConfigs, {
    selectors as machineAccessConfigSelectors,
} from './machineAccessConfigs';
import feedback, { selectors as feedbackSelectors } from './feedback';
import formMessages, { selectors as formMessageSelectors } from './formMessages';
import integrations, { selectors as integrationSelectors } from './integrations';
import invite, { selectors as inviteSelectors } from './invite';
import notifications, { selectors as notificationSelectors } from './notifications';
import policies, { selectors as policySelectors } from './policies/reducer';
import roles, { selectors as roleSelectors } from './roles';
import searchAutoComplete, { selectors as searchAutoCompleteSelectors } from './searchAutocomplete';
import serverResponseStatus, {
    selectors as serverResponseStatusSelectors,
} from './serverResponseStatus';

import loading, { selectors as loadingSelectors } from './loading';
import groups, { selectors as groupsSelectors } from './groups';
import centralCapabilities, {
    selectors as centralCapabilitiesSelectors,
} from './centralCapabilities';
import cloudSources, { selectors as cloudSourcesSelectors } from './cloudSources';

// Reducers

const appReducer = combineReducers({
    apiTokens,
    auth,
    machineAccessConfigs,
    feedback,
    formMessages,
    integrations,
    invite,
    notifications,
    policies,
    roles,
    searchAutoComplete,
    serverResponseStatus,
    loading,
    groups,
    centralCapabilities,
    cloudSources,
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

const getApp = (state) => state.app;
const getAPITokens = (state) => getApp(state).apiTokens;
const getAuth = (state) => getApp(state).auth;
const getMachineAccessConfigs = (state) => getApp(state).machineAccessConfigs;
const getFeedback = (state) => getApp(state).feedback;
const getFormMessages = (state) => getApp(state).formMessages;
const getIntegrations = (state) => getApp(state).integrations;
const getInvite = (state) => getApp(state).invite;
const getNotifications = (state) => getApp(state).notifications;
const getPolicies = (state) => getApp(state).policies;
const getRoles = (state) => getApp(state).roles;
const getSearchAutocomplete = (state) => getApp(state).searchAutoComplete;
const getServerResponseStatus = (state) => getApp(state).serverResponseStatus;
const getLoadingStatus = (state) => getApp(state).loading;

const getRuleGroups = (state) => getApp(state).groups;
const getCentralCapabilities = (state) => getApp(state).centralCapabilities;
const getCloudSources = (state) => getApp(state).cloudSources;

const boundSelectors = {
    ...bindSelectors(getAPITokens, apiTokenSelectors),
    ...bindSelectors(getAuth, authSelectors),
    ...bindSelectors(getMachineAccessConfigs, machineAccessConfigSelectors),
    ...bindSelectors(getFeedback, feedbackSelectors),
    ...bindSelectors(getFormMessages, formMessageSelectors),
    ...bindSelectors(getIntegrations, integrationSelectors),
    ...bindSelectors(getInvite, inviteSelectors),
    ...bindSelectors(getNotifications, notificationSelectors),
    ...bindSelectors(getPolicies, policySelectors),
    ...bindSelectors(getRoles, roleSelectors),
    ...bindSelectors(getSearchAutocomplete, searchAutoCompleteSelectors),
    ...bindSelectors(getServerResponseStatus, serverResponseStatusSelectors),
    ...bindSelectors(getLoadingStatus, loadingSelectors),

    ...bindSelectors(getRuleGroups, groupsSelectors),
    ...bindSelectors(getCentralCapabilities, centralCapabilitiesSelectors),
    ...bindSelectors(getCloudSources, cloudSourcesSelectors),
};

export const selectors = {
    ...boundSelectors,
};
