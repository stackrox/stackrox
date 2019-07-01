import { combineReducers } from 'redux';

// Action types
//-------------

export const types = {
    OPEN_WIZARD: 'policies/OPEN_WIZARD',
    CLOSE_WIZARD: 'policies/CLOSE_WIZARD',
    SET_POLICIES_ACTION: 'policies/SET_POLICIES_ACTION',
    CLOSE_DIALOGUE: 'policies/CLOSE_DIALOGUE'
};

// Actions
//---------

export const actions = {
    openWizard: () => ({ type: types.OPEN_WIZARD }),
    closeWizard: () => ({ type: types.CLOSE_WIZARD }),
    setPoliciesAction: policiesAction => ({ type: types.SET_POLICIES_ACTION, policiesAction }),
    closeDialogue: () => ({ type: types.CLOSE_DIALOGUE })
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const wizardOpen = (state = false, action) => {
    if (action.type === types.OPEN_WIZARD && state !== true) {
        return true;
    }
    if (action.type === types.CLOSE_WIZARD && state !== false) {
        return false;
    }
    return state;
};

const policiesAction = (state = '', action) => {
    if (action.type === types.SET_POLICIES_ACTION && !state) {
        return action.policiesAction;
    }
    if (action.type === types.CLOSE_DIALOGUE && state) {
        return '';
    }
    return state;
};

const reducer = combineReducers({
    wizardOpen,
    policiesAction
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const getWizardOpen = state => state.wizardOpen;

const getPoliciesAction = state => state.policiesAction;

export const selectors = {
    getWizardOpen,
    getPoliciesAction
};
