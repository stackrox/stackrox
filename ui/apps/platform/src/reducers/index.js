import { combineReducers } from 'redux';
import { reducer as formReducer } from 'redux-form';
import { connectRouter } from 'connected-react-router';

import bindSelectors from 'utils/bindSelectors';
import auth, { selectors as authSelectors } from './auth';
import feedback, { selectors as feedbackSelectors } from './feedback';
import invite, { selectors as inviteSelectors } from './invite';
import notifications, { selectors as notificationSelectors } from './notifications';
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

// Reducers

const appReducer = combineReducers({
    auth,
    feedback,
    invite,
    notifications,
    roles,
    searchAutoComplete,
    serverResponseStatus,
    loading,
    groups,
    centralCapabilities,
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
const getAuth = (state) => getApp(state).auth;
const getFeedback = (state) => getApp(state).feedback;
const getInvite = (state) => getApp(state).invite;
const getNotifications = (state) => getApp(state).notifications;
const getRoles = (state) => getApp(state).roles;
const getSearchAutocomplete = (state) => getApp(state).searchAutoComplete;
const getServerResponseStatus = (state) => getApp(state).serverResponseStatus;
const getLoadingStatus = (state) => getApp(state).loading;

const getRuleGroups = (state) => getApp(state).groups;
const getCentralCapabilities = (state) => getApp(state).centralCapabilities;

const boundSelectors = {
    ...bindSelectors(getAuth, authSelectors),
    ...bindSelectors(getFeedback, feedbackSelectors),
    ...bindSelectors(getInvite, inviteSelectors),
    ...bindSelectors(getNotifications, notificationSelectors),
    ...bindSelectors(getRoles, roleSelectors),
    ...bindSelectors(getSearchAutocomplete, searchAutoCompleteSelectors),
    ...bindSelectors(getServerResponseStatus, serverResponseStatusSelectors),
    ...bindSelectors(getLoadingStatus, loadingSelectors),

    ...bindSelectors(getRuleGroups, groupsSelectors),
    ...bindSelectors(getCentralCapabilities, centralCapabilitiesSelectors),
};

export const selectors = {
    ...boundSelectors,
};
