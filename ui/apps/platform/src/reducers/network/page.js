import { combineReducers } from 'redux';

import { timeWindows } from 'constants/timeWindows';

// Action types

export const types = {
    OPEN_SIDE_PANEL: 'network/OPEN_SIDE_PANEL',
    CLOSE_SIDE_PANEL: 'network/CLOSE_SIDE_PANEL',
    SET_NETWORK_ACTIVITY_TIME_WINDOW: 'network/SET_NETWORK_ACTIVITY_TIME_WINDOW',
};

// Actions

export const actions = {
    openSidePanel: () => ({ type: types.OPEN_SIDE_PANEL }),
    closeSidePanel: () => ({ type: types.CLOSE_SIDE_PANEL }),
    setNetworkActivityTimeWindow: (window) => ({
        type: types.SET_NETWORK_ACTIVITY_TIME_WINDOW,
        window,
    }),
};

// Reducers

const sidePanelOpen = (state = false, action) => {
    if (action.type === types.OPEN_SIDE_PANEL && state !== true) {
        return true;
    }
    if (action.type === types.CLOSE_SIDE_PANEL && state !== false) {
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
    sidePanelOpen,
    networkActivityTimeWindow,
});

// Selectors

const getSidePanelOpen = (state) => state.sidePanelOpen;
const getNetworkActivityTimeWindow = (state) => state.networkActivityTimeWindow;

export const selectors = {
    getSidePanelOpen,
    getNetworkActivityTimeWindow,
};

export default reducer;
