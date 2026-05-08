import isEmpty from 'lodash/isEmpty';

import type { ContainerSecurityContext } from 'types/deployment.proto';

// Sentence case (first word capitalized; proper nouns like SELinux excepted).
const securityContextFieldLabels: Record<keyof ContainerSecurityContext, string> = {
    privileged: 'Privileged',
    selinux: 'SELinux',
    dropCapabilities: 'Drop capabilities',
    addCapabilities: 'Add capabilities',
    readOnlyRootFilesystem: 'Read only root filesystem',
    seccompProfile: 'Seccomp profile',
    allowPrivilegeEscalation: 'Allow privilege escalation',
};

function isKeyofContainerSecurityContext(key: string): key is keyof ContainerSecurityContext {
    return key in securityContextFieldLabels;
}

function labelForSecurityContextKey(key: string): string {
    if (!isKeyofContainerSecurityContext(key)) {
        return key;
    }
    return securityContextFieldLabels[key];
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
            filteredValues.push([labelForSecurityContextKey(key), stringifiedArray]);
        } else if (
            // ensure any object value has at least one property that has a value
            typeof currentValue === 'object' &&
            currentValue && // guard against typeof NULL === 'object' bug
            Object.keys(currentValue).some((subKey) => currentValue[subKey])
        ) {
            try {
                const stringifiedObject = JSON.stringify(currentValue);
                filteredValues.push([labelForSecurityContextKey(key), stringifiedObject]);
            } catch {
                filteredValues.push([labelForSecurityContextKey(key), currentValue.toString()]); // fallback, if corrupt data prevent JSON parsing
            }
        } else if (!Array.isArray(currentValue) && (currentValue || currentValue === 0)) {
            // otherwise, check for truthy or numeric 0
            const stringifiedPrimitive = currentValue.toString();
            filteredValues.push([labelForSecurityContextKey(key), stringifiedPrimitive]);
        }
    });

    return filteredValues;
}
