import React, { ReactElement } from 'react';
import { Card, CardBody, DescriptionList, Divider, Flex, FlexItem } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import ContainerVolumes from './ContainerVolumes';
import ContainerSecrets from './ContainerSecrets';
import ContainerResources from './ContainerResources';
import ContainerImage from './ContainerImage';

function Commands({ command }) {
    return (
        <Flex direction={{ default: 'column' }}>
            {command.map((line) => (
                <FlexItem>{line}</FlexItem>
            ))}
        </Flex>
    );
}

function ContainerConfiguration({ container }): ReactElement {
    const { resources, volumes, secrets, config, image } = container;
    const { command, args } = config || {};
    return (
        <DescriptionList data-testid="container-configuration" isHorizontal>
            <ContainerImage image={image} />
            <Divider component="div" />
            {(command || args) && (
                <>
                    {command.length > 0 && (
                        <DescriptionListItem
                            term="Commands"
                            desc={<Commands command={command} />}
                            data-testid="commands"
                        />
                    )}
                    {args?.length > 0 && <DescriptionListItem term="Arguments" desc={args} />}
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

function ContainerConfigurations({ deployment }): ReactElement {
    return (
        <Card isFlat>
            <CardBody>
                {deployment?.containers?.length > 0
                    ? deployment.containers.map((container) => (
                          <ContainerConfiguration container={container} />
                      ))
                    : 'None'}
            </CardBody>
        </Card>
    );
}

export default ContainerConfigurations;
