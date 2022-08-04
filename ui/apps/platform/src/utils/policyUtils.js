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
