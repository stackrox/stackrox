import { combineReducers } from 'redux';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors,
} from 'reducers/pageSearch';

// Action types
//-------------

export const types = {
    ...searchTypes('network'),
};

const getNetworkSearchActions = getSearchActions('network');

const networkSearchActions = { ...getNetworkSearchActions };

// Network search should not show the following categories
const searchOptionExclusions = [
    'Cluster:',
    'Namespace:',
    'Namespace ID:',
    'Orchestrator Component:',
];
const filterSearchOptions = (options) =>
    options.filter((obj) => !searchOptionExclusions.includes(obj.value));
networkSearchActions.setNetworkSearchModifiers = (options) =>
    getNetworkSearchActions.setNetworkSearchModifiers(filterSearchOptions(options));
networkSearchActions.setNetworkSearchSuggestions = (options) =>
    getNetworkSearchActions.setNetworkSearchSuggestions(filterSearchOptions(options));

// Actions
//---------

export const actions = {
    ...networkSearchActions,
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const reducer = combineReducers({
    ...searchReducers('network'),
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

export const selectors = {
    ...getSearchSelectors('network'),
};
