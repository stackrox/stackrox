import { combineReducers } from 'redux';

// Action types
//-------------

export const types = {
    UPDATE_POLICY_DISABLED_STATE: 'policies/UPDATE_POLICY_DISABLED_STATE',
    SELECT_POLICY: 'policies/SELECT_POLICY',
    SELECT_POLICIES: 'policies/SELECT_POLICIES',
    SET_TABLE_PAGE: 'policies/SET_TABLE_PAGE'
};

// Actions
//-------------

export const actions = {
    updatePolicyDisabledState: ({ policyId, disabled }) => ({
        type: types.UPDATE_POLICY_DISABLED_STATE,
        policyId,
        disabled
    }),
    selectPolicyId: policyId => ({ type: types.SELECT_POLICY, policyId }),
    selectPolicyIds: policyIds => ({ type: types.SELECT_POLICIES, policyIds }),
    setTablePage: page => ({ type: types.SET_TABLE_PAGE, page })
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const selectedId = (state = '', action) => {
    if (action.type === types.SELECT_POLICY) {
        return action.policyId;
    }
    return state;
};

const selectedIds = (state = [], action) => {
    if (action.type === types.SELECT_POLICIES) {
        return action.policyIds;
    }
    return state;
};

const page = (state = 0, action) => {
    if (action.type === types.SET_TABLE_PAGE) {
        return action.page;
    }
    return state;
};

const reducer = combineReducers({
    selectedId,
    selectedIds,
    page
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/policies/reducer.js
//---------------------------------------------------------------------------------

const getSelectedPolicyId = state => state.selectedId;

const getSelectedPolicyIds = state => state.selectedIds;

const getTablePage = state => state.page;

export const selectors = {
    getSelectedPolicyId,
    getSelectedPolicyIds,
    getTablePage
};
