import React from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Text,
    TextVariants,
} from '@patternfly/react-core';

import { Deployment } from 'types/deployment.proto';

type SecurityContextProps = {
    deployment: Deployment;
};

function SecurityContext({ deployment }: SecurityContextProps) {
    return (
        <div>
            {deployment.containers.map((container) => {
                const securityContext = container?.securityContext;
                if (!securityContext || JSON.stringify(securityContext) === '{}') {
                    return (
                        <div>
                            <Text component={TextVariants.h3}>
                                Security context for container <strong>{container.name}</strong> not
                                detected
                            </Text>
                        </div>
                    );
                }
                return (
                    <div key={container.name} className="pf-u-mb-lg">
                        <Text
                            component={TextVariants.h3}
                            className="pf-u-font-size-lg pf-u-font-weight-bold"
                        >
                            Container: <em>{container.name}</em>
                        </Text>
                        <Divider component="div" />
                        <DescriptionList columnModifier={{ default: '2Col' }}>
                            {Object.keys(securityContext).map((key) => (
                                <DescriptionListGroup key={key}>
                                    <DescriptionListTerm>{key}</DescriptionListTerm>
                                    <DescriptionListDescription>
                                        <Text
                                            component={TextVariants.pre}
                                            className="pf-u-font-size-xs"
                                        >
                                            {JSON.stringify(securityContext[key], null, 2)}
                                        </Text>
                                    </DescriptionListDescription>
                                </DescriptionListGroup>
                            ))}
                        </DescriptionList>
                    </div>
                );
            })}
        </div>
    );
}

export default SecurityContext;
