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
    ...searchTypes('policies')
};

// Actions
//---------

export const actions = {
    ...getSearchActions('policies')
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const reducer = combineReducers({
    ...searchReducers('policies')
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

export const selectors = {
    ...getSearchSelectors('policies')
};
