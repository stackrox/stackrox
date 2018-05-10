import { combineReducers } from 'redux';
import mergeEntitiesById from 'utils/mergeEntitiesById';
import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_POLICIES: createFetchingActionTypes('policies/FETCH_POLICIES'),
    FETCH_POLICY_CATEGORIES: createFetchingActionTypes('policies/FETCH_POLICY_CATEGORIES'),
    ...searchTypes('policies')
};

// Actions

export const actions = {
    fetchPolicies: createFetchingActions(types.FETCH_POLICIES),
    fetchPolicyCategories: createFetchingActions(types.FETCH_POLICY_CATEGORIES),
    ...getSearchActions('policies')
};

// Reducers

const policies = (state = [], action) => {
    if (action.type === types.FETCH_POLICIES.SUCCESS) {
        return isEqual(action.response.policies, state) ? state : action.response.policies;
    }
    return state;
};

const policyCategories = (state = [], action) => {
    if (action.type === types.FETCH_POLICY_CATEGORIES.SUCCESS) {
        return isEqual(action.response.categories, state) ? state : action.response.categories;
    }
    return state;
};

const byId = (state = {}, action) => {
    if (action.response && action.response.entities && action.response.entities.policy) {
        return mergeEntitiesById(state, action.response.entities.policy);
    }
    return state;
};

const reducer = combineReducers({
    policyCategories,
    policies,
    byId,
    ...searchReducers('policies')
});

export default reducer;

// Selectors

const getPolicies = state => state.policies;
const getPoliciesById = state => state.byId;
const getPolicy = (state, id) => getPoliciesById(state)[id];
const getPolicyCategories = state => state.policyCategories;

export const selectors = {
    getPolicies,
    getPoliciesById,
    getPolicy,
    getPolicyCategories,
    ...getSearchSelectors('policies')
};
