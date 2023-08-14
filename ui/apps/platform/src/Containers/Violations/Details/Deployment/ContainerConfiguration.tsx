import React, { ReactElement } from 'react';
import { DescriptionList, Flex, FlexItem } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { Container } from 'types/deployment.proto';

import ContainerImage from './ContainerImage';
import ContainerResourcesDescriptionList from './ContainerResourcesDescriptionList';
import ContainerSecretDescriptionList from './ContainerSecretDescriptionList';
import ContainerVolumeDescriptionList from './ContainerVolumeDescriptionList';

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
        <DescriptionList isCompact isHorizontal>
            <ContainerImage image={image} />
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
                </>
            )}
            {!!resources && (
                <DescriptionListItem
                    term="Resources"
                    desc={
                        resources ? (
                            <ContainerResourcesDescriptionList resources={resources} />
                        ) : (
                            'None'
                        )
                    }
                />
            )}
            {!!volumes &&
                (volumes.length === 0 ? (
                    <DescriptionListItem term="volumes" desc="None" />
                ) : (
                    volumes.map((volume, i) => (
                        <DescriptionListItem
                            key={volume.name}
                            term={`volumes[${i}]`}
                            desc={<ContainerVolumeDescriptionList volume={volume} />}
                        />
                    ))
                ))}
            {!!secrets &&
                (secrets.length === 0 ? (
                    <DescriptionListItem term="secrets" desc="None" />
                ) : (
                    secrets.map((secret, i) => (
                        <DescriptionListItem
                            key={secret.name}
                            term={`secrets[${i}]`}
                            desc={<ContainerSecretDescriptionList secret={secret} />}
                        />
                    ))
                ))}
        </DescriptionList>
    );
}

export default ContainerConfiguration;
