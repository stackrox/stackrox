import { combineReducers } from 'redux';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';

// Action types
//-------------

export const types = {
    SET_DIALOGUE_STAGE: 'network/SET_DIALOGUE_STAGE',
    SET_NETWORK_NOTIFIERS: 'network/SET_NETWORK_NOTIFIERS',
    SEND_POLICY_MODIFICATION_NOTIFICATION: 'network/SEND_POLICY_MODIFICATION_NOTIFICATION'
};

// Actions
//---------

export const actions = {
    setNetworkDialogueStage: stage => ({ type: types.SET_DIALOGUE_STAGE, stage }),
    setNetworkNotifiers: notifierIds => ({ type: types.SET_NETWORK_NOTIFIERS, notifierIds }),
    notifyNetworkPolicyModification: () => ({ type: types.SEND_POLICY_MODIFICATION_NOTIFICATION })
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const networkDialogueStage = (state = dialogueStages.application, action) => {
    if (action.type === types.SET_DIALOGUE_STAGE) {
        return action.stage;
    }
    return state;
};

const selectedNetworkNotifiers = (state = null, action) => {
    if (action.type === types.SET_NETWORK_NOTIFIERS) {
        return action.notifierIds;
    }
    return state;
};

const reducer = combineReducers({
    networkDialogueStage,
    selectedNetworkNotifiers
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const getNetworkDialogueStage = state => state.networkDialogueStage;
const getNetworkNotifiers = state => state.selectedNetworkNotifiers;

export const selectors = {
    getNetworkDialogueStage,
    getNetworkNotifiers
};
