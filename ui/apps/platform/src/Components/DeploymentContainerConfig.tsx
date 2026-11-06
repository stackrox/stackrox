import { useState } from 'react';
import {
    ExpandableSection,
    Label,
    Split,
    SplitItem,
    Stack,
    StackItem,
} from '@patternfly/react-core';

import ContainerArgumentsInfo from 'Components/ContainerArgumentsInfo';
import ContainerCommandInfo from 'Components/ContainerCommandInfo';
import ContainerImageInfo from 'Components/ContainerImageInfo';
import type { ContainerImageSummary } from 'Components/ContainerImageInfo';
import ContainerResourcesInfo from 'Components/ContainerResourcesInfo';
import ContainerSecretsInfo from 'Components/ContainerSecretsInfo';
import ContainerVolumesInfo from 'Components/ContainerVolumesInfo';
import SecurityContext from 'Components/SecurityContext';
import type {
    ContainerConfig,
    ContainerResources,
    ContainerSecurityContext,
    ContainerType,
    ContainerVolume,
    EmbeddedSecret,
} from 'types/deployment.proto';

export type DeploymentContainerView = {
    id: string;
    name: string;
    type: ContainerType;
    image: ContainerImageSummary | null;
    resources: ContainerResources | null;
    volumes: ContainerVolume[];
    secrets: EmbeddedSecret[];
    config: Pick<ContainerConfig, 'args' | 'command'> | null;
    securityContext: ContainerSecurityContext | null;
};

type DeploymentContainerConfigProps = {
    container: DeploymentContainerView;
    getImageUrl: (imageId: string) => string;
};

function DeploymentContainerConfig({ container, getImageUrl }: DeploymentContainerConfigProps) {
    const [isExpanded, setIsExpanded] = useState(false);

    const onToggle = (_isExpanded: boolean) => {
        setIsExpanded(_isExpanded);
    };

    const toggleContent = (
        <Split hasGutter>
            <SplitItem>{container.name}</SplitItem>
            {container.type === 'INIT' && (
                <SplitItem>
                    <Label isCompact color="teal">
                        Init
                    </Label>
                </SplitItem>
            )}
        </Split>
    );

    return (
        <ExpandableSection
            toggleContent={toggleContent}
            onToggle={(_event, _isExpanded: boolean) => onToggle(_isExpanded)}
            isExpanded={isExpanded}
            isWidthLimited
            data-testid="deployment-container-config"
        >
            <Stack hasGutter>
                <StackItem>
                    <ContainerImageInfo image={container.image} getImageUrl={getImageUrl} />
                </StackItem>
                {container.resources && (
                    <StackItem>
                        <ContainerResourcesInfo resources={container.resources} />
                    </StackItem>
                )}
                <StackItem>
                    <ContainerVolumesInfo volumes={container.volumes} />
                </StackItem>
                <StackItem>
                    <ContainerSecretsInfo secrets={container.secrets} />
                </StackItem>
                <StackItem>
                    <ContainerArgumentsInfo args={container.config?.args ?? []} />
                </StackItem>
                <StackItem>
                    <ContainerCommandInfo command={container.config?.command ?? []} />
                </StackItem>
                {container.securityContext && (
                    <StackItem>
                        <SecurityContext securityContext={container.securityContext} />
                    </StackItem>
                )}
            </Stack>
        </ExpandableSection>
    );
}

export default DeploymentContainerConfig;
