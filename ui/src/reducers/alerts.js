import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    FETCH_GLOBAL_ALERT_COUNTS: createFetchingActionTypes('alerts/FETCH_GLOBAL_ALERT_COUNTS'),
    FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES: createFetchingActionTypes(
        'alerts/FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES'
    ),
    FETCH_ALERT_COUNTS_BY_CLUSTER: createFetchingActionTypes(
        'alerts/FETCH_ALERT_COUNTS_BY_CLUSTER'
    ),
    FETCH_ALERTS_BY_TIMESERIES: createFetchingActionTypes('alerts/FETCH_ALERTS_BY_TIMESERIES'),
    ...searchTypes('alerts')
};

// Actions

export const actions = {
    fetchGlobalAlertCounts: createFetchingActions(types.FETCH_GLOBAL_ALERT_COUNTS),
    fetchAlertCountsByPolicyCategories: createFetchingActions(
        types.FETCH_ALERT_COUNTS_BY_POLICY_CATEGORIES
    ),
    fetchAlertCountsByCluster: createFetchingActions(types.FETCH_ALERT_COUNTS_BY_CLUSTER),
    fetchAlertsByTimeseries: createFetchingActions(types.FETCH_ALERTS_BY_TIMESERIES),
    ...getSearchActions('alerts')
};

// Reducers

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
    globalAlertCounts,
    alertCountsByPolicyCategories,
    alertCountsByCluster,
    alertsByTimeseries,
    ...searchReducers('alerts')
});

export default reducer;

// Selectors

const getGlobalAlertCounts = state => state.globalAlertCounts;
const getAlertCountsByPolicyCategories = state => state.alertCountsByPolicyCategories;
const getAlertCountsByCluster = state => state.alertCountsByCluster;
const getAlertsByTimeseries = state => state.alertsByTimeseries;

export const selectors = {
    getGlobalAlertCounts,
    getAlertCountsByPolicyCategories,
    getAlertCountsByCluster,
    getAlertsByTimeseries,
    ...getSearchSelectors('alerts')
};
