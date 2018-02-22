import { combineReducers } from 'redux';

import mergeEntitiesById from 'utils/mergeEntitiesById';

// Reducers

const byId = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.policy) {
        return mergeEntitiesById(state, action.response.entities.policy);
    }
    return state;
};

const reducer = combineReducers({
    byId
});

export default reducer;

// Selectors

const getPoliciesById = state => state.byId;
const getPolicy = (state, id) => getPoliciesById(state)[id];

export const selectors = {
    getPoliciesById,
    getPolicy
};
