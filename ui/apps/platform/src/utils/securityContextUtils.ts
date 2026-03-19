import isEmpty from 'lodash/isEmpty';

import type { ContainerSecurityContext } from 'types/deployment.proto';

function toSentenceCase(str: string): string {
    // Convert camelCase to sentence case: addCapabilities -> Add capabilities
    const result = str
        .replace(/([A-Z])/g, ' $1')
        .replace(/^./, (char) => char.toUpperCase())
        .trim();
    return result;
}

export function getFilteredSecurityContextMap(
    securityContext: ContainerSecurityContext
): [string, string][] {
    // sort the keys of the security context, so any properties are shown in alpha order
    const sortedKeys = Object.keys(securityContext).sort();

    // build an array of only those properties that actually have values
    const filteredValues: [string, string][] = [];
    sortedKeys.forEach((key) => {
        const currentValue = securityContext[key];

        if (Array.isArray(currentValue) && !isEmpty(currentValue)) {
            // ensure any array has elements
            const stringifiedArray = currentValue.toString();
            filteredValues.push([toSentenceCase(key), stringifiedArray]);
        } else if (
            // ensure any object value has at least one property that has a value
            typeof currentValue === 'object' &&
            currentValue && // guard against typeof NULL === 'object' bug
            Object.keys(currentValue).some((subKey) => currentValue[subKey])
        ) {
            try {
                const stringifiedObject = JSON.stringify(currentValue);
                filteredValues.push([toSentenceCase(key), stringifiedObject]);
            } catch {
                filteredValues.push([toSentenceCase(key), currentValue.toString()]); // fallback, if corrupt data prevent JSON parsing
            }
        } else if (!Array.isArray(currentValue) && (currentValue || currentValue === 0)) {
            // otherwise, check for truthy or numeric 0
            const stringifiedPrimitive = currentValue.toString();
            filteredValues.push([toSentenceCase(key), stringifiedPrimitive]);
        }
    });

    return filteredValues;
}
