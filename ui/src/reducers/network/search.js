import { combineReducers } from 'redux';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types
//-------------

export const types = {
    ...searchTypes('network')
};

// Network search should not show the 'Cluster' category
const getNetworkSearchActions = getSearchActions('network');

const networkSearchActions = Object.assign({}, getNetworkSearchActions);

const filterSearchOptions = options => options.filter(obj => obj.value !== 'Cluster:');
networkSearchActions.setNetworkSearchModifiers = options =>
    getNetworkSearchActions.setNetworkSearchModifiers(filterSearchOptions(options));
networkSearchActions.setNetworkSearchSuggestions = options =>
    getNetworkSearchActions.setNetworkSearchSuggestions(filterSearchOptions(options));

// Actions
//---------

export const actions = {
    ...networkSearchActions
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const reducer = combineReducers({
    ...searchReducers('network')
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

export const selectors = {
    ...getSearchSelectors('network')
};
