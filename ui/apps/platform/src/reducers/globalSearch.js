import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors,
} from 'reducers/pageSearch';

// Action types

export const types = {
    FETCH_GLOBAL_SEARCH_RESULTS: createFetchingActionTypes(
        'globalsearch/FETCH_GLOBAL_SEARCH_RESULTS'
    ),
    TOGGLE_GLOBAL_SEARCH_VIEW: 'globalsearch/TOGGLE_GLOBAL_SEARCH_VIEW',
    SET_GLOBAL_SEARCH_CATEGORY: 'globalsearch/SET_GLOBAL_SEARCH_CATEGORY',
    ...searchTypes('global'),
};

// Actions

export const actions = {
    fetchGlobalSearchResults: createFetchingActions(types.FETCH_GLOBAL_SEARCH_RESULTS),
    toggleGlobalSearchView: () => ({
        type: types.TOGGLE_GLOBAL_SEARCH_VIEW,
    }),
    setGlobalSearchCategory: (category) => ({
        type: types.SET_GLOBAL_SEARCH_CATEGORY,
        category,
    }),
    ...getSearchActions('global'),
};

// Reducers

const globalSearchView = (state = false, action) => {
    if (action.type === types.TOGGLE_GLOBAL_SEARCH_VIEW) {
        return !state;
    }
    return state;
};

const globalSearchResults = (state = [], action) => {
    if (action.type === types.FETCH_GLOBAL_SEARCH_RESULTS.SUCCESS) {
        const results = action.response.results || [];
        return isEqual(results, state) ? state : results;
    }
    if (action.type === types.FETCH_GLOBAL_SEARCH_RESULTS.FAILURE) {
        const results = [];
        return isEqual(results, state) ? state : results;
    }
    return state;
};

const globalSearchCounts = (state = [], action) => {
    if (
        action.type === types.FETCH_GLOBAL_SEARCH_RESULTS.SUCCESS &&
        action.params.category === 'SEARCH_UNSET'
    ) {
        const counts = action.response.counts || [];
        return isEqual(counts, state) ? state : counts;
    }
    if (action.type === types.FETCH_GLOBAL_SEARCH_RESULTS.FAILURE) {
        const counts = [];
        return isEqual(counts, state) ? state : counts;
    }
    return state;
};

const globalSearchCategory = (state = 'SEARCH_UNSET', action) => {
    if (action.type === types.SET_GLOBAL_SEARCH_CATEGORY) {
        const { category } = action;
        return isEqual(category, state) ? state : category;
    }
    return state;
};

const reducer = combineReducers({
    globalSearchResults,
    globalSearchCounts,
    globalSearchView,
    globalSearchCategory,
    ...searchReducers('global'),
});

// Selectors

const getGlobalSearchResults = (state) => state.globalSearchResults;
const getGlobalSearchCounts = (state) => state.globalSearchCounts;
const getGlobalSearchView = (state) => state.globalSearchView;
const getGlobalSearchCategory = (state) => state.globalSearchCategory;

export const selectors = {
    getGlobalSearchResults,
    getGlobalSearchCounts,
    getGlobalSearchView,
    getGlobalSearchCategory,
    ...getSearchSelectors('global'),
};

export default reducer;
