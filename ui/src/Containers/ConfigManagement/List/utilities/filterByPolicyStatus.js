import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

const filterByPolicyStatus = (rows, state) => {
    if (!state) return rows;
    return rows.filter(row => {
        let passing = false;
        // policyStatus could be an object or a string
        if (row.policyStatus && row.policyStatus.failingPolicies) {
            const { length } = row.policyStatus.failingPolicies;
            if (!length) passing = true;
        } else if (row.policyStatus === 'pass') passing = true;
        if (state === SEARCH_OPTIONS.POLICY_STATUS.VALUES.PASS) return passing;
        if (state === SEARCH_OPTIONS.POLICY_STATUS.VALUES.FAIL) {
            return !passing;
        }
        return true;
    });
};

export default filterByPolicyStatus;
