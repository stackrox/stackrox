import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { ContainerVolume } from 'types/deployment.proto';

export type ContainerVolumeDescriptionListProps = {
    volume: ContainerVolume;
};

function ContainerVolumeDescriptionList({
    volume,
}: ContainerVolumeDescriptionListProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal>
            <DescriptionListItem term="Name" desc={volume.name} />
            {volume.source && <DescriptionListItem term="Source" desc={volume.source} />}
            <DescriptionListItem term="Destination" desc={volume.destination} />
            {volume.readOnly && <DescriptionListItem term="Read only" desc="true" />}
            <DescriptionListItem term="Type" desc={volume.type} />
            <DescriptionListItem term="Mount propagation" desc={volume.mountPropagation} />
        </DescriptionList>
    );
}

export default ContainerVolumeDescriptionList;
