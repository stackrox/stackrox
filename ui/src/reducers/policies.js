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
import { createSelector } from 'reselect';

// Action types

export const types = {
    FETCH_POLICIES: createFetchingActionTypes('policies/FETCH_POLICIES'),
    FETCH_POLICY: createFetchingActionTypes('policies/FETCH_POLICY'),
    FETCH_POLICY_CATEGORIES: createFetchingActionTypes('policies/FETCH_POLICY_CATEGORIES'),
    SET_POLICY_WIZARD_STATE: 'policies/SET_POLICY_WIZARD_STATE',
    UPDATE_POLICY: 'policies/UPDATE_POLICY',
    UPDATE_POLICY_DISABLED_STATE: 'policies/UPDATE_POLICY_DISABLED_STATE',
    REASSESS_POLICIES: 'policies/REASSESS_POLICIES',
    DELETE_POLICIES: 'policies/DELETE_POLICIES',
    ...searchTypes('policies')
};

// Actions

export const actions = {
    fetchPolicies: createFetchingActions(types.FETCH_POLICIES),
    fetchPolicy: createFetchingActions(types.FETCH_POLICY),
    fetchPolicyCategories: createFetchingActions(types.FETCH_POLICY_CATEGORIES),
    setPolicyWizardState: state => ({ type: types.SET_POLICY_WIZARD_STATE, state }),
    reassessPolicies: () => ({ type: types.REASSESS_POLICIES }),
    deletePolicies: policyIds => ({ type: types.DELETE_POLICIES, policyIds }),
    updatePolicy: policy => ({ type: types.UPDATE_POLICY, policy }),
    updatePolicyDisabledState: ({ policyId, disabled }) => ({
        type: types.UPDATE_POLICY_DISABLED_STATE,
        policyId,
        disabled
    }),
    ...getSearchActions('policies')
};

// Reducers

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

const defaultWizardState = {
    current: '',
    policy: null
};
const policyWizardState = (state = defaultWizardState, action) => {
    if (action.type === types.SET_POLICY_WIZARD_STATE) {
        const newState = Object.assign({}, state, action.state);
        return newState;
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
    policyWizardState,
    filteredIds,
    ...searchReducers('policies')
});

export default reducer;

// Selectors

const getPoliciesById = state => state.byId;
const getPolicy = (state, policyId) => getPoliciesById(state)[policyId];
const getPolicyCategories = state => state.policyCategories;
const getPolicyWizardState = state => state.policyWizardState;

const getPolicies = state => Object.values(getPoliciesById(state));
const getFilteredPoliciesIds = state => state.filteredIds;

const getFilteredPolicies = createSelector(
    [getPoliciesById, getFilteredPoliciesIds],
    (policies, ids) => (ids.policies && ids.policies.map(id => policies[id])) || []
);

export const selectors = {
    getPolicies,
    getPolicy,
    getPoliciesById,
    getPolicyCategories,
    getPolicyWizardState,
    getFilteredPoliciesIds,
    getFilteredPolicies,
    ...getSearchSelectors('policies')
};
