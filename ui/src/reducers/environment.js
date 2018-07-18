import { combineReducers } from 'redux';

import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    ...searchTypes('environment')
};

// Actions

export const actions = {
    ...getSearchActions('environment')
};

// Reducers

const reducer = combineReducers({
    ...searchReducers('environment')
});

// Selectors

export const selectors = {
    ...getSearchSelectors('environment')
};

export default reducer;
