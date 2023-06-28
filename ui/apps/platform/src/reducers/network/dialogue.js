import { combineReducers } from 'redux';

const dialogueStages = {
    closed: 'CLOSED',
    application: 'APPLICATION',
    notification: 'NOTIFICATION',
};

// Action types
//-------------

export const types = {
    SET_DIALOGUE_STAGE: 'network/SET_DIALOGUE_STAGE',
    SET_NETWORK_NOTIFIERS: 'network/SET_NETWORK_NOTIFIERS',
    SEND_POLICY_MODIFICATION_NOTIFICATION: 'network/SEND_POLICY_MODIFICATION_NOTIFICATION',
};

// Actions
//---------

export const actions = {
    setNetworkDialogueStage: (stage) => ({ type: types.SET_DIALOGUE_STAGE, stage }),
    setNetworkNotifiers: (notifierIds) => ({ type: types.SET_NETWORK_NOTIFIERS, notifierIds }),
    notifyNetworkPolicyModification: () => ({ type: types.SEND_POLICY_MODIFICATION_NOTIFICATION }),
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const networkDialogueStage = (state = dialogueStages.closed, action) => {
    if (action.type === types.SET_DIALOGUE_STAGE) {
        return action.stage;
    }
    return state;
};

const selectedNetworkNotifiers = (state = [], action) => {
    if (action.type === types.SET_NETWORK_NOTIFIERS) {
        return action.notifierIds;
    }
    return state;
};

const reducer = combineReducers({
    networkDialogueStage,
    selectedNetworkNotifiers,
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const getNetworkDialogueStage = (state) => state.networkDialogueStage;
const getNetworkNotifiers = (state) => state.selectedNetworkNotifiers;

export const selectors = {
    getNetworkDialogueStage,
    getNetworkNotifiers,
};
