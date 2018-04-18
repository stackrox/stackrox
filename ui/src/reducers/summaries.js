import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_SUMMARY_COUNTS: createFetchingActionTypes('summaries/FETCH_SUMMARY_COUNTS')
};

// Actions

export const actions = {
    fetchSummaryCounts: createFetchingActions(types.FETCH_SUMMARY_COUNTS)
};

// Reducers

const summaryCounts = (state = null, action) => {
    if (action.type === types.FETCH_SUMMARY_COUNTS.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const reducer = combineReducers({
    summaryCounts
});

export default reducer;

// Selectors

const getSummaryCounts = state => state.summaryCounts;

export const selectors = {
    getSummaryCounts
};
