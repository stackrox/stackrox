import { combineReducers } from 'redux';
import mergeEntitiesById from 'utils/mergeEntitiesById';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    ...searchTypes('policies')
};

// Actions

export const actions = {
    ...getSearchActions('policies')
};

// Reducers

const byId = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.policy) {
        return mergeEntitiesById(state, action.response.entities.policy);
    }
    return state;
};

const reducer = combineReducers({
    byId,
    ...searchReducers('policies')
});

export default reducer;

// Selectors

const getPoliciesById = state => state.byId;
const getPolicy = (state, id) => getPoliciesById(state)[id];

export const selectors = {
    getPoliciesById,
    getPolicy,
    ...getSearchSelectors('policies')
};
