import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import wizardStages from 'Containers/Network/Wizard/wizardStages';
import { types as deploymentTypes } from 'reducers/deployments';

// Action types
//-------------

export const types = {
    SET_WIZARD_STAGE: 'network/SET_WIZARD_STAGE',
    SET_POLICY_MODIFICATION_NAME: 'network/SET_POLICY_MODIFICATION_NAME'
};

// Actions
//---------

export const actions = {
    setNetworkWizardStage: stage => ({ type: types.SET_WIZARD_STAGE, stage }),
    setNetworkPolicyModificationName: name => ({
        type: types.SET_POLICY_MODIFICATION_NAME,
        name
    })
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const networkWizardStage = (state = wizardStages.details, action) => {
    if (action.type === types.SET_WIZARD_STAGE) {
        return action.stage;
    }
    return state;
};

const networkPolicyModificationName = (state = null, action) => {
    if (action.type === types.SET_POLICY_MODIFICATION_NAME) {
        return action.name;
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
    networkPolicyModificationName,
    selectedNodeDeployment
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const getNetworkWizardStage = state => state.networkWizardStage;
const getNetworkPolicyModificationName = state => state.networkPolicyModificationName;
const getNodeDeployment = state => state.selectedNodeDeployment;

export const selectors = {
    getNetworkWizardStage,
    getNetworkPolicyModificationName,
    getNodeDeployment
};
