import React, { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Stack,
    StackItem,
} from '@patternfly/react-core';

type ContainerSecurityContextMap = {
    privileged: { label: string };
    add_capabilities: { label: string };
    drop_capabilities: { label: string };
};

const getSecurityContext = (container): ContainerSecurityContextMap | null => {
    if (!container.securityContext) {
        return null;
    }
    const { privileged, add_capabilities, drop_capabilities } = container.securityContext; // eslint-disable-line
    return { privileged, add_capabilities, drop_capabilities };
};

const SecurityContext = ({ deployment }): ReactElement => {
    let containerResult: ReactElement | ReactElement[];
    if (deployment.containers) {
        const containers = deployment.containers
            .filter((container) => !!container.securityContext)
            .map((container) => {
                const securityContext = getSecurityContext(container);
                if (!securityContext || JSON.stringify(securityContext) === '{}') {
                    return null;
                }
                return (
                    <Stack hasGutter key={container.toString()}>
                        <StackItem>
                            <DescriptionList columnModifier={{ default: '2Col' }}>
                                {Object.keys(securityContext).map((key) => (
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>{key}</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {securityContext[key]}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                ))}
                            </DescriptionList>
                        </StackItem>
                    </Stack>
                );
            });
        containerResult = containers.length ? (
            containers
        ) : (
            <span className="py-3 font-600 italic">None</span>
        );
    } else {
        containerResult = <span className="py-3 font-600 italic">None</span>;
    }
    return <div className="flex h-full px-3">{containerResult}</div>;
};

export default SecurityContext;
