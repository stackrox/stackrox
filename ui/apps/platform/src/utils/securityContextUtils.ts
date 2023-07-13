import isEmpty from 'lodash/isEmpty';

import { ContainerSecurityContext } from 'types/deployment.proto';

export function getFilteredSecurityContextMap(
    securityContext: ContainerSecurityContext
): Map<string, string> {
    // sort the keys of the security context, so any properties are shown in alpha order
    const sortedKeys = Object.keys(securityContext).sort();

    // build a map of only those properties that actually have values
    const filteredValues = new Map<string, string>();
    sortedKeys.forEach((key) => {
        const currentValue = securityContext[key];

        if (Array.isArray(currentValue) && !isEmpty(currentValue)) {
            // ensure any array has elements
            const stringifiedArray = currentValue.toString();
            filteredValues.set(key, stringifiedArray);
        } else if (
            // ensure any object value has at least one property that has a value
            typeof currentValue === 'object' &&
            currentValue && // guard against typeof NULL === 'object' bug
            Object.keys(currentValue).some((subKey) => currentValue[subKey])
        ) {
            try {
                const stringifiedObject = JSON.stringify(currentValue);
                filteredValues.set(key, stringifiedObject);
            } catch (err) {
                filteredValues.set(key, currentValue.toString()); // fallback, if corrupt data prevent JSON parsing
            }
        } else if (!Array.isArray(currentValue) && (currentValue || currentValue === 0)) {
            // otherwise, check for truthy or numeric 0
            const stringifiedPrimitive = currentValue.toString();
            filteredValues.set(key, stringifiedPrimitive);
        }
    });

    return filteredValues;
}
