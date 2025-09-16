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

type ContainerConfigurationDescriptionListProps = {
    container: Container;
    vulnMgmtBasePath: string;
};

function ContainerConfigurationDescriptionList({
    container,
    vulnMgmtBasePath,
}: ContainerConfigurationDescriptionListProps): ReactElement {
    const { resources, volumes, secrets, config, image } = container;
    const { command, args } = config || {};
    return (
        <DescriptionList isCompact isHorizontal>
            <ContainerImage image={image} vulnMgmtBasePath={vulnMgmtBasePath} />
            <DescriptionListItem
                term="Commands"
                desc={command?.length > 0 ? <MultilineDescription descArr={command} /> : 'None'}
                aria-label="Commands"
            />
            <DescriptionListItem
                term="Arguments"
                desc={args?.length > 0 ? <MultilineDescription descArr={args} /> : 'None'}
                aria-label="Arguments"
            />
            <DescriptionListItem
                term="Resources"
                desc={
                    resources ? <ContainerResourcesDescriptionList resources={resources} /> : 'None'
                }
            />
            {volumes == null || volumes.length === 0 ? (
                <DescriptionListItem term="volumes" desc="None" />
            ) : (
                volumes.map((volume, i) => (
                    <DescriptionListItem
                        key={volume.name}
                        term={`volumes[${i}]`}
                        desc={<ContainerVolumeDescriptionList volume={volume} />}
                    />
                ))
            )}
            {secrets == null || secrets.length === 0 ? (
                <DescriptionListItem term="secrets" desc="None" />
            ) : (
                secrets.map((secret, i) => (
                    <DescriptionListItem
                        key={`${secret.name}_${secret.path}`}
                        term={`secrets[${i}]`}
                        desc={<ContainerSecretDescriptionList secret={secret} />}
                    />
                ))
            )}
        </DescriptionList>
    );
}

export default ContainerConfigurationDescriptionList;
