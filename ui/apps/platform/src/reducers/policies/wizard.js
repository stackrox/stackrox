import { combineReducers } from 'redux';

// Action types
//-------------

export const types = {
    SET_WIZARD_POLICY: 'policies/SET_WIZARD_POLICY',
    SET_WIZARD_POLICY_DISABLED: 'policies/SET_WIZARD_POLICY_DISABLED',
};

// Actions
//---------

export const actions = {
    setWizardPolicy: (policy) => ({ type: types.SET_WIZARD_POLICY, policy }),
    setWizardPolicyDisabled: (disabled) => ({ type: types.SET_WIZARD_POLICY_DISABLED, disabled }),
};

// Helpers
//--------

const setPolicy = (state, policy) => {
    const newState = { ...state, policy };
    newState.isNew = !newState.policy.id;
    return newState;
};

const setPolicyDisabled = (state, disabled) => {
    const newState = {};
    newState.policy = { ...state.policy, disabled };
    newState.isNew = state.isNew;
    return newState;
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const wizardPolicy = (state = { isNew: false, policy: null }, action) => {
    if (action.type === types.SET_WIZARD_POLICY) {
        return setPolicy(state, action.policy);
    }
    if (action.type === types.SET_WIZARD_POLICY_DISABLED) {
        return setPolicyDisabled(state, action.disabled);
    }
    return state;
};

const reducer = combineReducers({
    wizardPolicy,
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const getWizardIsNew = (state) => state.wizardPolicy.isNew;

const getWizardPolicy = (state) => state.wizardPolicy.policy;

export const selectors = {
    getWizardIsNew,
    getWizardPolicy,
};
