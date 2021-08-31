import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';

function ContainerResources({ resources }): ReactElement {
    return (
        <DescriptionList isHorizontal>
            <DescriptionListItem term="CPU Request (cores)" desc={resources.cpuCoresRequest} />
            <DescriptionListItem term="CPU Limit (cores)" desc={resources.cpuCoresLimit} />
            <DescriptionListItem term="Memory Request (MB)" desc={resources.memoryMbRequest} />
            <DescriptionListItem term="Memory Limit (MB)" desc={resources.memoryMbLimit} />
        </DescriptionList>
    );
}

export default ContainerResources;
