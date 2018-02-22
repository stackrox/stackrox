import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import mergeEntitiesById from 'utils/mergeEntitiesById';

// Action types

export const types = {
    SELECT_VIOLATED_POLICY: 'alerts/SELECT_VIOLATED_POLICY',
    FETCH_ALERTS_BY_POLICY: createFetchingActionTypes('alerts/FETCH_ALERTS_BY_POLICY'),
    FETCH_ALERT_NUMS_BY_POLICY: createFetchingActionTypes('alerts/FETCH_ALERT_NUMS_BY_POLICY'),
    FETCH_ALERT: createFetchingActionTypes('alerts/FETCH_ALERT')
};

// Actions

export const actions = {
    selectViolatedPolicy: policyId => ({ type: types.SELECT_VIOLATED_POLICY, policyId }),
    fetchAlertsByPolicy: createFetchingActions(types.FETCH_ALERTS_BY_POLICY),
    fetchAlertNumsByPolicy: createFetchingActions(types.FETCH_ALERT_NUMS_BY_POLICY),
    fetchAlert: createFetchingActions(types.FETCH_ALERT)
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

const reducer = combineReducers({
    byId,
    numsByPolicy,
    selectedViolatedPolicy,
    alertsByPolicy
});

export default reducer;

// Selectors

const getAlertsById = state => state.byId;
const getAlert = (state, id) => getAlertsById(state)[id];
const getAlertNumsByPolicy = state => state.numsByPolicy;
const getSelectedViolatedPolicyId = state => state.selectedViolatedPolicy;
const getAlertsByPolicy = state => state.alertsByPolicy;

export const selectors = {
    getAlertsById,
    getAlertNumsByPolicy,
    getAlert,
    getSelectedViolatedPolicyId,
    getAlertsByPolicy
};
