import { combineReducers } from 'redux';
import sidepanelStages from 'Containers/Network/SidePanel/sidepanelStages';

// Action types
//-------------

export const types = {
    SET_SIDE_PANEL_STAGE: 'network/SET_SIDE_PANEL_STAGE',
    SET_POLICY_MODIFICATION: 'network/SET_POLICY_MODIFICATION',
    SET_POLICY_MODIFICATION_NAME: 'network/SET_POLICY_MODIFICATION_NAME',
    SET_POLICY_MODIFICATION_SOURCE: 'network/SET_POLICY_MODIFICATION_SOURCE',
    SET_POLICY_MODIFICATION_STATE: 'network/SET_POLICY_MODIFICATION_STATE',
    SET_POLICY_EXCLUDE_PORTS_PROTOCOLS_STATE: 'network/SET_POLICY_EXCLUDE_PORTS_PROTOCOLS_STATE',
    GENERATE_NETWORK_POLICY_MODIFICATION: 'network/GENERATE_NETWORK_POLICY_MODIFICATION',
    LOAD_ACTIVE_NETWORK_POLICY_MODIFICATION: 'network/LOAD_ACTIVE_NETWORK_POLICY_MODIFICATION',
    LOAD_UNDO_NETWORK_POLICY_MODIFICATION: 'network/LOAD_UNDO_NETWORK_POLICY_MODIFICATION',
};

// Actions
//---------

export const actions = {
    setSidePanelStage: (stage) => ({ type: types.SET_SIDE_PANEL_STAGE, stage }),
    setNetworkPolicyModification: (modification) => ({
        type: types.SET_POLICY_MODIFICATION,
        modification,
    }),
    setNetworkPolicyModificationName: (name) => ({
        type: types.SET_POLICY_MODIFICATION_NAME,
        name,
    }),
    setNetworkPolicyModificationSource: (source) => ({
        type: types.SET_POLICY_MODIFICATION_SOURCE,
        source,
    }),
    setNetworkPolicyModificationState: (state) => ({
        type: types.SET_POLICY_MODIFICATION_STATE,
        state,
    }),
    setNetworkPolicyExcludePortsProtocolsState: (state) => ({
        type: types.SET_POLICY_EXCLUDE_PORTS_PROTOCOLS_STATE,
        state,
    }),
    generateNetworkPolicyModification: () => ({
        type: types.GENERATE_NETWORK_POLICY_MODIFICATION,
    }),
    loadActiveNetworkPolicyModification: () => ({
        type: types.LOAD_ACTIVE_NETWORK_POLICY_MODIFICATION,
    }),
    loadUndoNetworkPolicyModification: () => ({
        type: types.LOAD_UNDO_NETWORK_POLICY_MODIFICATION,
    }),
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const networkSidePanelStage = (state = sidepanelStages.details, action) => {
    if (action.type === types.SET_SIDE_PANEL_STAGE) {
        return action.stage;
    }
    return state;
};

const networkPolicyModification = (state = null, action) => {
    if (action.type === types.SET_POLICY_MODIFICATION) {
        return action.modification;
    }
    return state;
};

const networkPolicyModificationName = (state = '', action) => {
    if (action.type === types.SET_POLICY_MODIFICATION_NAME) {
        return action.name;
    }
    return state;
};

const networkPolicyModificationSource = (state = null, action) => {
    if (action.type === types.SET_POLICY_MODIFICATION_SOURCE) {
        return action.source;
    }
    return state;
};

const networkPolicyModificationState = (state = 'SUCCESS', action) => {
    if (action.type === types.SET_POLICY_MODIFICATION_STATE) {
        return action.state;
    }
    return state;
};

const networkPolicyExcludePortsProtocolsState = (state = false, action) => {
    if (action.type === types.SET_POLICY_EXCLUDE_PORTS_PROTOCOLS_STATE) {
        return action.state;
    }
    return state;
};

const reducer = combineReducers({
    networkSidePanelStage,
    networkPolicyModification,
    networkPolicyModificationName,
    networkPolicyModificationSource,
    networkPolicyModificationState,
    networkPolicyExcludePortsProtocolsState,
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const getSidePanelStage = (state) => state.networkSidePanelStage;
const getNetworkPolicyModification = (state) => state.networkPolicyModification;
const getNetworkPolicyModificationName = (state) => state.networkPolicyModificationName;
const getNetworkPolicyModificationSource = (state) => state.networkPolicyModificationSource;
const getNetworkPolicyModificationState = (state) => state.networkPolicyModificationState;
const getNetworkPolicyExcludePortsProtocolsState = (state) =>
    state.networkPolicyExcludePortsProtocolsState;

export const selectors = {
    getSidePanelStage,
    getNetworkPolicyModification,
    getNetworkPolicyModificationName,
    getNetworkPolicyModificationSource,
    getNetworkPolicyModificationState,
    getNetworkPolicyExcludePortsProtocolsState,
};
