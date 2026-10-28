import { useState } from 'react';
import {
    ExpandableSection,
    Label,
    Split,
    SplitItem,
    Stack,
    StackItem,
} from '@patternfly/react-core';

import type {
    Container,
    ContainerConfig,
    ContainerImage,
    ContainerResources,
    ContainerSecurityContext,
    ContainerVolume,
    EmbeddedSecret,
} from 'types/deployment.proto';
import type { ImageName } from 'types/image.proto';
import ContainerImageInfo from 'Components/ContainerImageInfo';
import ContainerResourcesInfo from 'Components/ContainerResourcesInfo';
import ContainerVolumesInfo from 'Components/ContainerVolumesInfo';
import ContainerSecretsInfo from 'Components/ContainerSecretsInfo';
import ContainerArgumentsInfo from 'Components/ContainerArgumentsInfo';
import ContainerCommandInfo from 'Components/ContainerCommandInfo';
import SecurityContext from 'Components/SecurityContext';

export type DeploymentContainerConfigData = Pick<Container, 'id' | 'name' | 'type'> & {
    image:
        | (Pick<ContainerImage, 'id' | 'idV2' | 'notPullable'> & {
              name: ImageName | null;
          })
        | null;
    resources: ContainerResources | null;
    volumes: ContainerVolume[];
    secrets: EmbeddedSecret[];
    config: Pick<ContainerConfig, 'args' | 'command'> | null;
    securityContext: ContainerSecurityContext | null;
};

type DeploymentContainerConfigProps = {
    container: DeploymentContainerConfigData;
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
