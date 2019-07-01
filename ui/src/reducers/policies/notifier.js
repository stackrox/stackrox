import { combineReducers } from 'redux';

// Action types
//-------------

export const types = {
    SET_POLICY_NOTIFIERS: 'policies/SET_POLICY_NOTIFIERS'
};

// Actions
//---------

export const actions = {
    setPolicyNotifiers: notifierIds => ({ type: types.SET_POLICY_NOTIFIERS, notifierIds })
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const selectedPolicyNotifiers = (state = [], action) => {
    if (action.type === types.SET_POLICY_NOTIFIERS) {
        return action.notifierIds;
    }
    return state;
};

const reducer = combineReducers({
    selectedPolicyNotifiers
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const getPolicyNotifiers = state => state.selectedPolicyNotifiers;

export const selectors = {
    getPolicyNotifiers
};
