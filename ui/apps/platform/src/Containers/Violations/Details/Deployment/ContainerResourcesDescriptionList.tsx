import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { ContainerResources } from 'types/deployment.proto';

export type ContainerResourcesDescriptionListProps = {
    resources: ContainerResources;
};

function ContainerResourcesDescriptionList({
    resources,
}: ContainerResourcesDescriptionListProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal>
            <DescriptionListItem term="CPU request (cores)" desc={resources.cpuCoresRequest} />
            <DescriptionListItem term="CPU limit (cores)" desc={resources.cpuCoresLimit} />
            <DescriptionListItem term="Memory request (MB)" desc={resources.memoryMbRequest} />
            <DescriptionListItem term="Memory limit (MB)" desc={resources.memoryMbLimit} />
        </DescriptionList>
    );
}

export default ContainerResourcesDescriptionList;
