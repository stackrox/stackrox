import { combineReducers } from 'redux';

// Action types
//-------------

export const types = {
    OPEN_WIZARD: 'policies/OPEN_WIZARD',
    CLOSE_WIZARD: 'policies/CLOSE_WIZARD',
    OPEN_DIALOGUE: 'policies/OPEN_DIALOGUE',
    CLOSE_DIALOGUE: 'policies/CLOSE_DIALOGUE'
};

// Actions
//---------

export const actions = {
    openWizard: () => ({ type: types.OPEN_WIZARD }),
    closeWizard: () => ({ type: types.CLOSE_WIZARD }),
    openDialogue: () => ({ type: types.OPEN_DIALOGUE }),
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

const dialogueOpen = (state = false, action) => {
    if (action.type === types.OPEN_DIALOGUE && state !== true) {
        return true;
    }
    if (action.type === types.CLOSE_DIALOGUE && state !== false) {
        return false;
    }
    return state;
};

const reducer = combineReducers({
    wizardOpen,
    dialogueOpen
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const getWizardOpen = state => state.wizardOpen;

const getDialogueOpen = state => state.dialogueOpen;

export const selectors = {
    getWizardOpen,
    getDialogueOpen
};
