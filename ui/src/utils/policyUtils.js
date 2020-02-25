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
            [severities.LOW_SEVERITY]: 0
        }
    );
    return {
        critical: counts.CRITICAL_SEVERITY,
        high: counts.HIGH_SEVERITY,
        medium: counts.MEDIUM_SEVERITY,
        low: counts.LOW_SEVERITY
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
