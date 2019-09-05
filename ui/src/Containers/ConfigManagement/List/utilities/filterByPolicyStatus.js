import { capitalize } from 'lodash';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

const filterByPolicyStatus = (rows, state) => {
    if (!state || !rows) return rows;
    const policyState = capitalize(state);
    return rows.filter(row => {
        let passing = false;
        // policyStatus could be an object or a string
        if (row.policyStatus && row.policyStatus.failingPolicies) {
            const { length } = row.policyStatus.failingPolicies;
            if (!length) passing = true;
        } else if (row.policyStatus === 'pass') passing = true;
        if (policyState === SEARCH_OPTIONS.POLICY_STATUS.VALUES.PASS) return passing;
        if (policyState === SEARCH_OPTIONS.POLICY_STATUS.VALUES.FAIL) {
            return !passing;
        }
        return true;
    });
};

export default filterByPolicyStatus;
