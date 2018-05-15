import { combineReducers } from 'redux';

import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    ...searchTypes('dashboard')
};

// Actions

// Dashboard search should only be able to show the 'Cluster' category
const getDashboardSearchActions = getSearchActions('dashboard');
const dashboardSearchActions = Object.assign({}, getDashboardSearchActions);
const filterClusterOption = options => options.filter(obj => obj.value === 'Cluster:');
dashboardSearchActions.setDashboardSearchModifiers = options =>
    getDashboardSearchActions.setDashboardSearchModifiers(filterClusterOption(options));
dashboardSearchActions.setDashboardSearchSuggestions = options =>
    getDashboardSearchActions.setDashboardSearchSuggestions(filterClusterOption(options));

export const actions = {
    ...dashboardSearchActions
};

const reducer = combineReducers({
    ...searchReducers('dashboard')
});

export default reducer;

// Selectors

export const selectors = {
    ...getSearchSelectors('dashboard')
};
