// Action types
//-------------

export const types = {
    UPDATE_POLICY_DISABLED_STATE: 'policies/UPDATE_POLICY_DISABLED_STATE',
};

// Actions
//-------------

export const actions = {
    updatePolicyDisabledState: ({ policyId, disabled }) => ({
        type: types.UPDATE_POLICY_DISABLED_STATE,
        policyId,
        disabled,
    }),
};
