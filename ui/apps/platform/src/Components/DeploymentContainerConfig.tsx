import React, { useState } from 'react';
import { ExpandableSection, Stack, StackItem } from '@patternfly/react-core';

import { Container } from 'types/deployment.proto';
import ContainerImageInfo from 'Components/ContainerImageInfo';
import ContainerResourcesInfo from 'Components/ContainerResouresInfo';
import ContainerVolumesInfo from 'Components/ContainerVolumesInfo';
import ContainerSecretsInfo from 'Components/ContainerSecretsInfo';
import ContainerArgumentsInfo from 'Components/ContainerArgumentsInfo';
import ContainerCommandInfo from 'Components/ContainerCommandInfo';
import SecurityContext from 'Components/SecurityContext';

type DeploymentContainerConfigProps = {
    container: Container;
};

function DeploymentContainerConfig({ container }: DeploymentContainerConfigProps) {
    const [isExpanded, setIsExpanded] = useState(false);

    const onToggle = (_isExpanded: boolean) => {
        setIsExpanded(_isExpanded);
    };

    const toggleText = container.name;

    return (
        <ExpandableSection
            toggleText={toggleText}
            onToggle={onToggle}
            isExpanded={isExpanded}
            displaySize="large"
            isWidthLimited
            data-testid="deployment-container-config"
        >
            <Stack hasGutter>
                <StackItem>
                    <ContainerImageInfo image={container.image} />
                </StackItem>
                <StackItem>
                    <ContainerResourcesInfo resources={container.resources} />
                </StackItem>
                <StackItem>
                    <ContainerVolumesInfo volumes={container.volumes} />
                </StackItem>
                <StackItem>
                    <ContainerSecretsInfo secrets={container.secrets} />
                </StackItem>
                <StackItem>
                    <ContainerArgumentsInfo args={container.config.args} />
                </StackItem>
                <StackItem>
                    <ContainerCommandInfo command={container.config.command} />
                </StackItem>
                <StackItem>
                    <SecurityContext securityContext={container.securityContext} />
                </StackItem>
            </Stack>
        </ExpandableSection>
    );
}

export default DeploymentContainerConfig;
