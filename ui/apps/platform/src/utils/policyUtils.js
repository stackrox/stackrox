import cloneDeep from 'lodash/cloneDeep';

import removeEmptyFieldsDeep from 'utils/removeEmptyFieldsDeep';
import { severities } from 'constants/severities';

export function getPolicySeverityCounts(failingPolicies) {
    const counts = failingPolicies.reduce(
        (acc, curr) => {
            if (curr && curr.severity && Object.keys(severities).includes(curr.severity)) {
                acc[curr.severity] += 1;
            }
            return acc;
        },
        {
            [severities.CRITICAL_SEVERITY]: 0,
            [severities.HIGH_SEVERITY]: 0,
            [severities.MEDIUM_SEVERITY]: 0,
            [severities.LOW_SEVERITY]: 0,
        }
    );
    return {
        critical: counts.CRITICAL_SEVERITY,
        high: counts.HIGH_SEVERITY,
        medium: counts.MEDIUM_SEVERITY,
        low: counts.LOW_SEVERITY,
    };
}

/**
 * sort deployments by most severe policy violations first
 *   - any violation at a higher level trumps any number of violations at a lower level
 *   - ties at one level get sorted by the next closest different lower level
 *
 * @param   {array}  deployments  list of deployments, each with policySeverityCounts object
 *
 * @return  {array}               a copy of the suppliced list, sorted
 */
export function sortDeploymentsByPolicyViolations(deployments = []) {
    // create a copy of the array
    const copiedArray = deployments.concat();

    const sortedDeployments = copiedArray.sort(comparePoliciesCounts);

    return sortedDeployments;
}

/**
 * this function does the opposite of a normal sort callback:
 *   it sorts in reverse order, so that the larger number of failures comes first
 *
 *   this saves the step of having to reverse the array after it is sorted
 *
 * @param   {object}  a  one deployment with a policySeverityCounts object
 * @param   {object}  b  another deployment with a policySeverityCounts object
 *
 * @return  {number}     0 if all counts are equal,
 *                       -1 if deployment A has more severe violations than deployment B
 *                       1 if deployment B has more severe violations than deployment A
 */
function comparePoliciesCounts(a, b) {
    let result;
    result = compareCounts(a.policySeverityCounts.critical, b.policySeverityCounts.critical);
    if (result !== 0) {
        return result;
    }
    result = compareCounts(a.policySeverityCounts.high, b.policySeverityCounts.high);
    if (result !== 0) {
        return result;
    }
    result = compareCounts(a.policySeverityCounts.medium, b.policySeverityCounts.medium);
    if (result !== 0) {
        return result;
    }
    result = compareCounts(a.policySeverityCounts.low, b.policySeverityCounts.low);
    if (result !== 0) {
        return result;
    }
    return 0;
}

/**
 * this function does the opposite of a normal sort callback:
 *   it sorts in reverse order, so that the larger number of failures comes first
 *
 *   this saves the step of having to reverse the array after it is sorted
 *
 * @param   {number}  a  number of violations for one object
 * @param   {number}  b  number of violations for another object
 *
 * @return  {number}     0 if counts are equal,
 *                       -1 if A is greater B
 *                       1 B is greater than A
 */
function compareCounts(a, b) {
    if (!a && !b) {
        return 0;
    }
    if (a && !b) {
        return -1;
    }
    if (b && !a) {
        return 1;
    }
    if (a > b) {
        return -1;
    }
    if (b > a) {
        return 1;
    }

    return 0;
}

/**
 * convert all policy values to strings, to match BPL API requirements
 * *
 * @param   {object}  policy  a Policy Wizard-form policy
 *
 * @return  {object}          that policy, with all values transformed to
 */
export function transformPolicyCriteriaValuesToStrings(policy) {
    const newPolicySections = !policy?.policySections
        ? []
        : policy.policySections.map((section) => {
              const newPolicyGroups = !section?.policyGroups
                  ? []
                  : section?.policyGroups.map((group) => {
                        let newValues = null;
                        if (group?.values?.length) {
                            newValues = group.values.map((valueObj) => {
                                const currentVal = valueObj.value;
                                let newVal = currentVal;
                                if (typeof currentVal !== 'string') {
                                    newVal = currentVal.toString();
                                }
                                return { ...valueObj, value: newVal };
                            });
                        }
                        return newValues ? { ...group, values: newValues } : group;
                    });
              return { ...section, policyGroups: newPolicyGroups };
          });
    const transformedPolicy = { ...policy, policySections: newPolicySections };

    return transformedPolicy;
}

/**
 * extract the names of excluded scopes from the bespoken exclusion list
 * returned by the single policy GraphQL resolver
 *
 * @param   {array}  scopes          (see below for object shape)
 * @param   {string}  exclusionType  'deployment' or 'image'
 *
 * @return  {string}                 comma-separated list of scope names
 *
 * example data:
 * [
 *     {
 *          deployment: {
 *              name: 'central',
 *              scope: null,
 *          },
 *          image: null,
 *      },
 *      {
 *          deployment: null,
 *          image: {
 *              name: 'docker.io/library/mysql:5',
 *          },
 *      },
 * ]
 */
export function getExcludedNamesByType(scopes, exclusionType) {
    // first, does the exclusion have an object of the specified type?
    const filteredScopes = scopes.filter((scope) => Boolean(scope[exclusionType]));

    const names = filteredScopes.reduce((list, scope) => {
        return list.concat(scope[exclusionType].name);
    }, []);

    return names.join(', ');
}

/**
 * Deletes empty fields from the policy object taking into account
 * API & UI specific fields.
 *
 * @param {object} policy a policy object
 * @return {object} cleaned-up deep copy of a policy object w/o empty fields
 */
export function removeEmptyPolicyFields(policy) {
    const cleanedPolicyCopy = removeEmptyFieldsDeep(policy);

    if (policy.policySections) {
        // never remove policySections as it's always expected to be present on a policy object
        cleanedPolicyCopy.policySections = cloneDeep(policy.policySections);
    }

    // The following fields are not used if they have falsy values,
    //   but those still returned from the API,
    //   so we have to filter them out separately
    //   Note: `readOnlyRootFs` is not in this list, because its only allowed value is `false`
    const exceptionFields = ['whitelistEnabled'];
    exceptionFields.forEach((fieldName) => {
        if (!cleanedPolicyCopy[fieldName]) {
            delete cleanedPolicyCopy[fieldName];
        }
    });

    return cleanedPolicyCopy;
}
