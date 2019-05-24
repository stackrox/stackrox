import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import pick from 'lodash/pick';
import { createSelector } from 'reselect';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import { createPollingActionTypes, createPollingActions } from 'utils/pollingReduxRoutines';
import mergeEntitiesById from 'utils/mergeEntitiesById';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    SELECT_VIOLATED_POLICY: 'alerts/SELECT_VIOLATED_POLICY',
    FETCH_ALERTS: createFetchingActionTypes('alerts/FETCH_ALERTS'),
    POLL_ALERTS: createPollingActionTypes('alerts/POLL_ALERTS'),
    FETCH_ALERT: createFetchingActionTypes('alerts/FETCH_ALERT'),
    FETCH_GLOBAL_ALERT_COUNTS: createFetchingActionTypes('alerts/FETCH_GLOBAL_ALERT_COUNTS'),
    FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES: createFetchingActionTypes(
        'alerts/FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES'
    ),
    FETCH_ALERT_COUNTS_BY_CLUSTER: createFetchingActionTypes(
        'alerts/FETCH_ALERT_COUNTS_BY_CLUSTER'
    ),
    FETCH_ALERTS_BY_TIMESERIES: createFetchingActionTypes('alerts/FETCH_ALERTS_BY_TIMESERIES'),
    WHITELIST_DEPLOYMENT: createFetchingActionTypes('alerts/WHITELIST_DEPLOYMENT'),
    WHITELIST_DEPLOYMENTS: createFetchingActionTypes('alerts/WHITELIST_DEPLOYMENTS'),
    RESOLVE_ALERTS: 'alerts/RESOLVE_ALERTS',
    ...searchTypes('alerts')
};

// Actions

export const actions = {
    selectViolatedPolicy: policyId => ({ type: types.SELECT_VIOLATED_POLICY, policyId }),
    fetchAlerts: createFetchingActions(types.FETCH_ALERTS),
    pollAlerts: createPollingActions(types.POLL_ALERTS),
    fetchAlert: createFetchingActions(types.FETCH_ALERT),
    fetchGlobalAlertCounts: createFetchingActions(types.FETCH_GLOBAL_ALERT_COUNTS),
    fetchAlertCountsByPolicyCategories: createFetchingActions(
        types.FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES
    ),
    fetchAlertCountsByCluster: createFetchingActions(types.FETCH_ALERT_COUNTS_BY_CLUSTER),
    fetchAlertsByTimeseries: createFetchingActions(types.FETCH_ALERTS_BY_TIMESERIES),
    whitelistDeployment: createFetchingActions(types.WHITELIST_DEPLOYMENT),
    whitelistDeployments: createFetchingActions(types.WHITELIST_DEPLOYMENTS),
    resolveAlerts: (alertIds, whitelist) => ({ type: types.RESOLVE_ALERTS, alertIds, whitelist }),
    ...getSearchActions('alerts')
};

// Reducers

const byId = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.alert) {
        const alertsById = action.response.entities.alert;
        const newState = mergeEntitiesById(state, alertsById);
        if (
            action.type === types.FETCH_ALERTS.SUCCESS &&
            (!action.params || !action.params.options || !action.params.options.length)
        ) {
            // fetched all alerts without any filter/search options, leave only those alerts
            const onlyExisting = pick(newState, Object.keys(alertsById));
            return isEqual(onlyExisting, state) ? state : onlyExisting;
        }
        return newState;
    }
    return state;
};

const filteredIds = (state = [], action) => {
    if (action.type === types.FETCH_ALERTS.SUCCESS) {
        const alertIds = action.response.result ? action.response.result.alerts : [];
        return isEqual(alertIds, state) ? state : alertIds;
    }
    return state;
};

const globalAlertCounts = (state = [], action) => {
    if (action.type === types.FETCH_GLOBAL_ALERT_COUNTS.SUCCESS) {
        const { groups } = action.response;
        return isEqual(groups, state) ? state : groups;
    }
    return state;
};

const alertCountsByPolicyCategories = (state = [], action) => {
    if (action.type === types.FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES.SUCCESS) {
        const { groups } = action.response;
        return isEqual(groups, state) ? state : groups;
    }
    return state;
};

const alertCountsByCluster = (state = [], action) => {
    if (action.type === types.FETCH_ALERT_COUNTS_BY_CLUSTER.SUCCESS) {
        const { groups } = action.response;
        return isEqual(groups, state) ? state : groups;
    }
    return state;
};

const alertsByTimeseries = (state = [], action) => {
    if (action.type === types.FETCH_ALERTS_BY_TIMESERIES.SUCCESS) {
        const { clusters } = action.response;
        return isEqual(clusters, state) ? state : clusters;
    }
    return state;
};

const reducer = combineReducers({
    byId,
    filteredIds,
    globalAlertCounts,
    alertCountsByPolicyCategories,
    alertCountsByCluster,
    alertsByTimeseries,
    ...searchReducers('alerts')
});

export default reducer;

// Selectors

const getAlertsById = state => state.byId;
const getAlerts = state => Object.values(getAlertsById(state));
const getFilteredIds = state => state.filteredIds;
const getAlert = (state, id) => getAlertsById(state)[id];
const getGlobalAlertCounts = state => state.globalAlertCounts;
const getAlertCountsByPolicyCategories = state => state.alertCountsByPolicyCategories;
const getAlertCountsByCluster = state => state.alertCountsByCluster;
const getAlertsByTimeseries = state => state.alertsByTimeseries;
const getFilteredAlertsById = createSelector(
    [getAlertsById, getFilteredIds],
    (alerts, ids) => {
        const alertsObj = {};
        ids.forEach(id => {
            alertsObj[id] = alerts[id];
        });
        return alertsObj;
    }
);
const getFilteredAlerts = state => Object.values(getFilteredAlertsById(state));

export const selectors = {
    getAlertsById,
    getAlerts,
    getFilteredIds,
    getAlert,
    getGlobalAlertCounts,
    getAlertCountsByPolicyCategories,
    getAlertCountsByCluster,
    getAlertsByTimeseries,
    getFilteredAlerts,
    getFilteredAlertsById,
    ...getSearchSelectors('alerts')
};
