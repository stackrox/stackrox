import { combineReducers } from 'redux';
import mergeEntitiesById from 'utils/mergeEntitiesById';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import { createSelector } from 'reselect';

// Action types
//-------------

export const types = {
    FETCH_POLICIES: createFetchingActionTypes('policies/FETCH_POLICIES'),
    FETCH_POLICY: createFetchingActionTypes('policies/FETCH_POLICY'),
    FETCH_POLICY_CATEGORIES: createFetchingActionTypes('policies/FETCH_POLICY_CATEGORIES'),
    UPDATE_POLICY: 'policies/UPDATE_POLICY',
    REASSESS_POLICIES: 'policies/REASSESS_POLICIES',
    DELETE_POLICIES: 'policies/DELETE_POLICIES'
};

// Actions
//---------

export const actions = {
    fetchPolicies: createFetchingActions(types.FETCH_POLICIES),
    fetchPolicy: createFetchingActions(types.FETCH_POLICY),
    fetchPolicyCategories: createFetchingActions(types.FETCH_POLICY_CATEGORIES),
    reassessPolicies: () => ({ type: types.REASSESS_POLICIES }),
    deletePolicies: policyIds => ({ type: types.DELETE_POLICIES, policyIds }),
    updatePolicy: policy => ({ type: types.UPDATE_POLICY, policy })
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const policyCategories = (state = [], action) => {
    if (action.type === types.FETCH_POLICY_CATEGORIES.SUCCESS) {
        return isEqual(action.response.categories, state) ? state : action.response.categories;
    }
    return state;
};

const byId = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.policy) {
        return mergeEntitiesById(state, action.response.entities.policy, true);
    }
    return state;
};

const filteredIds = (state = {}, action) => {
    if (action.type === types.FETCH_POLICIES.SUCCESS) {
        return isEqual(action.response.result, state) ? state : action.response.result;
    }
    return state;
};

const reducer = combineReducers({
    policyCategories,
    byId,
    filteredIds
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const getPoliciesById = state => state.byId;
const getFilteredPolicyIds = state => state.filteredIds;
const getPolicyCategories = state => state.policyCategories;

const getPolicy = (state, policyId) => getPoliciesById(state)[policyId];
const getPolicies = state => Object.values(getPoliciesById(state));
const getFilteredPolicies = createSelector(
    [getPoliciesById, getFilteredPolicyIds],
    (policies, ids) => (ids.policies && ids.policies.map(id => policies[id])) || []
);

export const selectors = {
    getPolicies,
    getPolicy,
    getPoliciesById,
    getPolicyCategories,
    getFilteredPolicyIds,
    getFilteredPolicies
};
