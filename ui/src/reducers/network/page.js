import { combineReducers } from 'redux';

// Action types

export const types = {
    OPEN_WIZARD: 'network/OPEN_WIZARD',
    CLOSE_WIZARD: 'network/CLOSE_WIZARD'
};

// Actions

export const actions = {
    openNetworkWizard: () => ({ type: types.OPEN_WIZARD }),
    closeNetworkWizard: () => ({ type: types.CLOSE_WIZARD })
};

// Reducers

const wizardOpen = (state = false, action) => {
    if (action.type === types.OPEN_WIZARD && state !== true) {
        return true;
    }
    if (action.type === types.CLOSE_WIZARD && state !== false) {
        return false;
    }
    return state;
};

const reducer = combineReducers({
    wizardOpen
});

// Selectors

const getNetworkWizardOpen = state => state.wizardOpen;

export const selectors = {
    getNetworkWizardOpen
};

export default reducer;
