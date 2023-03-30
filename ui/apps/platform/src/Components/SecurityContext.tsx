import React from 'react';
import { isEmpty } from 'lodash';
import {
    Card,
    CardBody,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    EmptyState,
} from '@patternfly/react-core';

import { ContainerSecurityContext } from 'types/deployment.proto';

type SecurityContextProps = {
    securityContext: ContainerSecurityContext;
};

function SecurityContext({ securityContext }: SecurityContextProps) {
    // build a map of only those properties that actually have values

    // sort the keys of the prop, so that the map is in alpha order
    const sortedKeys = Object.keys(securityContext).sort();

    //
    const filteredValues = new Map();
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

    return (
        <Card>
            <CardTitle>Security context</CardTitle>
            <CardBody className="pf-u-background-color-200 pf-u-pt-xl pf-u-mx-lg pf-u-mb-lg">
                {filteredValues.size > 0 ? (
                    <DescriptionList columnModifier={{ default: '2Col' }} isCompact>
                        {Array.from(filteredValues.entries()).map(([key, value]) => {
                            return (
                                <DescriptionListGroup>
                                    <DescriptionListTerm>{key}</DescriptionListTerm>
                                    <DescriptionListDescription>{value}</DescriptionListDescription>
                                </DescriptionListGroup>
                            );
                        })}
                    </DescriptionList>
                ) : (
                    <EmptyState>No container security context</EmptyState>
                )}
            </CardBody>
        </Card>
    );
}

export default SecurityContext;
