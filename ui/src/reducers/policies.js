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
    SET_POLICY_WIZARD_STATE: 'policies/SET_POLICY_WIZARD_STATE',
    UPDATE_POLICY: 'policies/UPDATE_POLICY',
    REASSESS_POLICIES: 'policies/REASSESS_POLICIES',
    ...searchTypes('policies')
};

// Actions

export const actions = {
    fetchPolicies: createFetchingActions(types.FETCH_POLICIES),
    fetchPolicyCategories: createFetchingActions(types.FETCH_POLICY_CATEGORIES),
    setPolicyWizardState: state => ({ type: types.SET_POLICY_WIZARD_STATE, state }),
    reassessPolicies: () => ({ type: types.REASSESS_POLICIES }),
    updatePolicy: policy => ({ type: types.UPDATE_POLICY, policy }),
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
        return mergeEntitiesById(state, action.response.entities.policy);
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

const reducer = combineReducers({
    policyCategories,
    byId,
    policyWizardState,
    ...searchReducers('policies')
});

export default reducer;

// Selectors

const getPoliciesById = state => state.byId;
const getPolicy = (state, id) => getPoliciesById(state)[id];
const getPolicyCategories = state => state.policyCategories;
const getPolicyWizardState = state => state.policyWizardState;

export const selectors = {
    getPoliciesById,
    getPolicy,
    getPolicyCategories,
    getPolicyWizardState,
    ...getSearchSelectors('policies')
};
