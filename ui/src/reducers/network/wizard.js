import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import wizardStages from 'Containers/Network/Wizard/wizardStages';
import { types as deploymentTypes } from 'reducers/deployments';

// Action types
//-------------

export const types = {
    SET_WIZARD_STAGE: 'network/SET_WIZARD_STAGE',
    SET_POLICY_MODIFICATION: 'network/SET_POLICY_MODIFICATION',
    SET_POLICY_MODIFICATION_NAME: 'network/SET_POLICY_MODIFICATION_NAME',
    SET_POLICY_MODIFICATION_SOURCE: 'network/SET_POLICY_MODIFICATION_SOURCE',
    SET_POLICY_MODIFICATION_STATE: 'network/SET_POLICY_MODIFICATION_STATE',
    GENERATE_NETWORK_POLICY_MODIFICATION: 'network/GENERATE_NETWORK_POLICY_MODIFICATION',
    LOAD_ACTIVE_NETWORK_POLICY_MODIFICATION: 'network/LOAD_ACTIVE_NETWORK_POLICY_MODIFICATION',
    LOAD_UNDO_NETWORK_POLICY_MODIFICATION: 'network/LOAD_UNDO_NETWORK_POLICY_MODIFICATION'
};

// Actions
//---------

export const actions = {
    setNetworkWizardStage: stage => ({ type: types.SET_WIZARD_STAGE, stage }),
    setNetworkPolicyModification: modification => ({
        type: types.SET_POLICY_MODIFICATION,
        modification
    }),
    setNetworkPolicyModificationName: name => ({
        type: types.SET_POLICY_MODIFICATION_NAME,
        name
    }),
    setNetworkPolicyModificationSource: source => ({
        type: types.SET_POLICY_MODIFICATION_SOURCE,
        source
    }),
    setNetworkPolicyModificationState: state => ({
        type: types.SET_POLICY_MODIFICATION_STATE,
        state
    }),
    generateNetworkPolicyModification: () => ({ type: types.GENERATE_NETWORK_POLICY_MODIFICATION }),
    loadActiveNetworkPolicyModification: () => ({
        type: types.LOAD_ACTIVE_NETWORK_POLICY_MODIFICATION
    }),
    loadUndoNetworkPolicyModification: () => ({
        type: types.LOAD_UNDO_NETWORK_POLICY_MODIFICATION
    })
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const networkWizardStage = (state = wizardStages.details, action) => {
    if (action.type === types.SET_WIZARD_STAGE) {
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

const selectedNodeDeployment = (state = {}, action) => {
    if (action.response && action.response.entities) {
        const { entities, result } = action.response;
        if (entities && entities.deployment && result) {
            const deploymentsById = Object.assign({}, entities.deployment[result]);
            if (action.type === deploymentTypes.FETCH_DEPLOYMENT.SUCCESS) {
                return isEqual(deploymentsById, state) ? state : deploymentsById;
            }
        }
    }
    return state;
};

const reducer = combineReducers({
    networkWizardStage,
    networkPolicyModification,
    networkPolicyModificationName,
    networkPolicyModificationSource,
    networkPolicyModificationState,
    selectedNodeDeployment
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const getNetworkWizardStage = state => state.networkWizardStage;
const getNetworkPolicyModification = state => state.networkPolicyModification;
const getNetworkPolicyModificationName = state => state.networkPolicyModificationName;
const getNetworkPolicyModificationSource = state => state.networkPolicyModificationSource;
const getNetworkPolicyModificationState = state => state.networkPolicyModificationState;
const getNodeDeployment = state => state.selectedNodeDeployment;

export const selectors = {
    getNetworkWizardStage,
    getNetworkPolicyModification,
    getNetworkPolicyModificationName,
    getNetworkPolicyModificationSource,
    getNetworkPolicyModificationState,
    getNodeDeployment
};
