import React, { ReactElement } from 'react';
import { Card, CardBody, DescriptionList, Divider, Flex, FlexItem } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { Container, Deployment } from 'types/deployment.proto';
import ContainerVolumes from './ContainerVolumes';
import ContainerSecrets from './ContainerSecrets';
import ContainerResources from './ContainerResources';
import ContainerImage from './ContainerImage';

function MultilineDescription({ descArr }) {
    return (
        <Flex direction={{ default: 'column' }}>
            {descArr.map((desc, idx) => (
                // eslint-disable-next-line react/no-array-index-key
                <div key={idx}>
                    <FlexItem>{desc}</FlexItem>
                </div>
            ))}
        </Flex>
    );
}

type ContainerConfigurationProps = {
    container: Container;
};

function ContainerConfiguration({ container }: ContainerConfigurationProps): ReactElement {
    const { resources, volumes, secrets, config, image } = container;
    const { command, args } = config || {};
    return (
        <DescriptionList isHorizontal>
            <ContainerImage image={image} />
            <Divider component="div" />
            {(command?.length > 0 || args?.length > 0) && (
                <>
                    {command.length > 0 && (
                        <DescriptionListItem
                            term="Commands"
                            desc={<MultilineDescription descArr={command} />}
                            aria-label="Commands"
                        />
                    )}
                    {args?.length > 0 && (
                        <DescriptionListItem
                            term="Arguments"
                            desc={<MultilineDescription descArr={args} />}
                            aria-label="Arguments"
                        />
                    )}
                    <Divider component="div" />
                </>
            )}
            {!!resources && (
                <>
                    <DescriptionListItem
                        term="Resources"
                        desc={resources ? <ContainerResources resources={resources} /> : 'None'}
                    />
                    <Divider component="div" />
                </>
            )}
            {!!volumes && (
                <>
                    <DescriptionListItem
                        term="Volumes"
                        desc={volumes?.length > 0 ? <ContainerVolumes volumes={volumes} /> : 'None'}
                    />
                    <Divider component="div" />
                </>
            )}
            {!!secrets && (
                <DescriptionListItem
                    term="Secrets"
                    desc={secrets?.length > 0 ? <ContainerSecrets secrets={secrets} /> : 'None'}
                />
            )}
        </DescriptionList>
    );
}

export type ContainerConfigurationsProps = {
    deployment: Deployment | null;
};

function ContainerConfigurations({ deployment }: ContainerConfigurationsProps): ReactElement {
    const containers = deployment?.containers || [];
    return (
        <Card isFlat aria-label="Container configuration">
            <CardBody>
                {containers.length > 0
                    ? containers.map((container, idx) => (
                          // eslint-disable-next-line react/no-array-index-key
                          <ContainerConfiguration container={container} key={idx} />
                      ))
                    : 'None'}
            </CardBody>
        </Card>
    );
}

export default ContainerConfigurations;
