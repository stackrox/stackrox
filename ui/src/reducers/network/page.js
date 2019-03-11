import { combineReducers } from 'redux';

import timeWindows from 'constants/timeWindows';

// Action types

export const types = {
    OPEN_WIZARD: 'network/OPEN_WIZARD',
    CLOSE_WIZARD: 'network/CLOSE_WIZARD',
    OPEN_DIALOGUE: 'network/OPEN_DIALOGUE',
    CLOSE_DIALOGUE: 'network/CLOSE_DIALOGUE',
    SET_NETWORK_ACTIVITY_TIME_WINDOW: 'network/SET_NETWORK_ACTIVITY_TIME_WINDOW'
};

// Actions

export const actions = {
    openNetworkWizard: () => ({ type: types.OPEN_WIZARD }),
    closeNetworkWizard: () => ({ type: types.CLOSE_WIZARD }),
    openNetworkDialogue: () => ({ type: types.OPEN_DIALOGUE }),
    closeNetworkDialogue: () => ({ type: types.CLOSE_DIALOGUE }),
    setNetworkActivityTimeWindow: window => ({
        type: types.SET_NETWORK_ACTIVITY_TIME_WINDOW,
        window
    })
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

const dialogueOpen = (state = false, action) => {
    if (action.type === types.OPEN_DIALOGUE && state !== true) {
        return true;
    }
    if (action.type === types.CLOSE_DIALOGUE && state !== false) {
        return false;
    }
    return state;
};

const networkActivityTimeWindow = (state = timeWindows[0], action) => {
    if (action.type === types.SET_NETWORK_ACTIVITY_TIME_WINDOW) {
        return action.window;
    }
    return state;
};

const reducer = combineReducers({
    wizardOpen,
    dialogueOpen,
    networkActivityTimeWindow
});

// Selectors

const getNetworkWizardOpen = state => state.wizardOpen;
const getNetworDialogueOpen = state => state.dialogueOpen;
const getNetworkActivityTimeWindow = state => state.networkActivityTimeWindow;

export const selectors = {
    getNetworkWizardOpen,
    getNetworDialogueOpen,
    getNetworkActivityTimeWindow
};

export default reducer;
