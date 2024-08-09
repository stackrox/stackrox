import { Policy } from 'types/policy.proto';

// TODO - According to the usage of this function, it does nothing. The only case where a conversion
//        will occur is when a runtime value does not match the expected TS value in the Policy object.
//        The only place this should be possible at the moment is when called via VM 1.0 "Save to Policy" functionality,
//        where the code remains in plain JS. Once VM 1.0 Dashboard is removed, this function can be deleted as well.
/**
 *
 * convert all policy values to strings, to match BPL API requirements
 *
 * @param policy a Policy Wizard-form policy
 * @return policy, with all values transformed to
 */
export function transformPolicyCriteriaValuesToStrings(policy: Policy) {
    const newPolicySections = !policy?.policySections
        ? []
        : policy.policySections.map((section) => {
              const newPolicyGroups = !section?.policyGroups
                  ? []
                  : section?.policyGroups.map((group) => {
                        const newValues = group?.values?.length
                            ? group.values.map((valueObj) => {
                                  const currentVal = valueObj.value;
                                  let newVal = currentVal;
                                  if (typeof currentVal !== 'string') {
                                      newVal = String(currentVal);
                                  }
                                  return { ...valueObj, value: newVal };
                              })
                            : null;
                        return newValues ? { ...group, values: newValues } : group;
                    });
              return { ...section, policyGroups: newPolicyGroups };
          });
    const transformedPolicy = { ...policy, policySections: newPolicySections };

    return transformedPolicy;
}
