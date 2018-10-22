import { combineReducers } from 'redux';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';

// Action types
//-------------

export const types = {
    SET_WIZARD_STAGE: 'policies/SET_WIZARD_STAGE',
    SET_WIZARD_POLICY: 'policies/SET_WIZARD_POLICY',
    SET_WIZARD_DRY_RUN: 'policies/SET_WIZARD_DRY_RUN',
    SET_WIZARD_POLICY_DISABLED: 'policies/SET_WIZARD_POLICY_DISABLED'
};

// Actions
//---------

export const actions = {
    setWizardStage: stage => ({ type: types.SET_WIZARD_STAGE, stage }),
    setWizardPolicy: policy => ({ type: types.SET_WIZARD_POLICY, policy }),
    setWizardDryRun: dryRun => ({ type: types.SET_WIZARD_DRY_RUN, dryRun }),
    setWizardPolicyDisabled: disabled => ({ type: types.SET_WIZARD_POLICY_DISABLED, disabled })
};

// Helpers
//--------

const setPolicy = (state, policy) => {
    const newState = Object.assign({}, state, { policy });
    newState.isNew = !newState.policy.id;
    return newState;
};

const setPolicyDisabled = (state, disabled) => {
    const newState = {};
    newState.policy = Object.assign({}, state.policy, { disabled });
    newState.isNew = state.isNew;
    return newState;
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const wizardStage = (state = wizardStages.details, action) => {
    if (action.type === types.SET_WIZARD_STAGE) {
        return action.stage;
    }
    return state;
};

const wizardPolicy = (state = { isNew: false, policy: null }, action) => {
    if (action.type === types.SET_WIZARD_POLICY) {
        return setPolicy(state, action.policy);
    }
    if (action.type === types.SET_WIZARD_POLICY_DISABLED) {
        return setPolicyDisabled(state, action.disabled);
    }
    return state;
};

const wizardDryRun = (state = null, action) => {
    if (action.type === types.SET_WIZARD_DRY_RUN) {
        return action.dryRun;
    }
    return state;
};

const reducer = combineReducers({
    wizardStage,
    wizardPolicy,
    wizardDryRun
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const getWizardStage = state => state.wizardStage;

const getWizardIsNew = state => state.wizardPolicy.isNew;

const getWizardPolicy = state => state.wizardPolicy.policy;

const getWizardDryRun = state => state.wizardDryRun;

export const selectors = {
    getWizardStage,
    getWizardIsNew,
    getWizardPolicy,
    getWizardDryRun
};
