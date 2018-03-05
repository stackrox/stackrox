import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import mergeEntitiesById from 'utils/mergeEntitiesById';

// Action types

export const types = {
    SELECT_VIOLATED_POLICY: 'alerts/SELECT_VIOLATED_POLICY',
    FETCH_ALERTS_BY_POLICY: createFetchingActionTypes('alerts/FETCH_ALERTS_BY_POLICY'),
    FETCH_ALERT_NUMS_BY_POLICY: createFetchingActionTypes('alerts/FETCH_ALERT_NUMS_BY_POLICY'),
    FETCH_ALERT: createFetchingActionTypes('alerts/FETCH_ALERT'),
    FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES: createFetchingActionTypes(
        'alerts/FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES'
    ),
    FETCH_ALERT_COUNTS_BY_CLUSTER: createFetchingActionTypes(
        'alerts/FETCH_ALERT_COUNTS_BY_CLUSTER'
    ),
    FETCH_ALERTS_BY_TIMESERIES: createFetchingActionTypes('alerts/FETCH_ALERTS_BY_TIMESERIES')
};

// Actions

export const actions = {
    selectViolatedPolicy: policyId => ({ type: types.SELECT_VIOLATED_POLICY, policyId }),
    fetchAlertsByPolicy: createFetchingActions(types.FETCH_ALERTS_BY_POLICY),
    fetchAlertNumsByPolicy: createFetchingActions(types.FETCH_ALERT_NUMS_BY_POLICY),
    fetchAlert: createFetchingActions(types.FETCH_ALERT),
    fetchAlertCountsByPolicyCategories: createFetchingActions(
        types.FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES
    ),
    fetchAlertCountsByCluster: createFetchingActions(types.FETCH_ALERT_COUNTS_BY_CLUSTER),
    fetchAlertsByTimeseries: createFetchingActions(types.FETCH_ALERTS_BY_TIMESERIES)
};

// Reducers

const byId = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.alert) {
        return mergeEntitiesById(state, action.response.entities.alert);
    }
    return state;
};

const numsByPolicy = (state = [], action) => {
    if (action.type === types.FETCH_ALERT_NUMS_BY_POLICY.SUCCESS) {
        const { alertsByPolicies } = action.response.result;
        return isEqual(alertsByPolicies, state) ? state : alertsByPolicies;
    }
    return state;
};

const selectedViolatedPolicy = (state = null, action) => {
    if (action.type === types.SELECT_VIOLATED_POLICY) {
        return action.policyId || null;
    }
    if (state && action.type === types.FETCH_ALERT_NUMS_BY_POLICY.SUCCESS) {
        const { alertsByPolicies } = action.response.result;
        // received a new list of violated policies and it doesn't contain selected policy: unselect
        if (!alertsByPolicies.map(alertNum => alertNum.policy).includes(state)) return null;
    }
    return state;
};

const alertsByPolicy = (state = {}, action) => {
    if (action.type === types.FETCH_ALERTS_BY_POLICY.SUCCESS) {
        const { alerts } = action.response.result;
        const { params: policyId } = action;
        return isEqual(state[policyId], alerts) ? state : { ...state, [policyId]: alerts };
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
        const { alertEvents } = action.response;
        return isEqual(alertEvents, state) ? state : alertEvents;
    }
    return state;
};

const reducer = combineReducers({
    byId,
    numsByPolicy,
    selectedViolatedPolicy,
    alertsByPolicy,
    alertCountsByPolicyCategories,
    alertCountsByCluster,
    alertsByTimeseries
});

export default reducer;

// Selectors

const getAlertsById = state => state.byId;
const getAlert = (state, id) => getAlertsById(state)[id];
const getAlertNumsByPolicy = state => state.numsByPolicy;
const getSelectedViolatedPolicyId = state => state.selectedViolatedPolicy;
const getAlertsByPolicy = state => state.alertsByPolicy;
const getAlertCountsByPolicyCategories = state => state.alertCountsByPolicyCategories;
const getAlertCountsByCluster = state => state.alertCountsByCluster;
const getAlertsByTimeseries = state => state.alertsByTimeseries;

export const selectors = {
    getAlertsById,
    getAlertNumsByPolicy,
    getAlert,
    getSelectedViolatedPolicyId,
    getAlertsByPolicy,
    getAlertCountsByPolicyCategories,
    getAlertCountsByCluster,
    getAlertsByTimeseries
};
