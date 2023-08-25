import React from 'react';
import { Flex, Tooltip } from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

import { SourceType } from '../Tables/table.utils';

export type ComponentLocationTdProps = {
    location: string;
    source: SourceType;
};

function ComponentLocationTd({ location, source }: ComponentLocationTdProps) {
    return (
        <Flex spaceItems={{ default: 'spaceItemsXs' }} alignItems={{ default: 'alignItemsCenter' }}>
            <span>{location || 'N/A'}</span>
            {source === 'OS' && (
                <Tooltip content="Location is unavailable for operating system packages">
                    <InfoCircleIcon color="var(--pf-global--info-color--100)" />
                </Tooltip>
            )}
        </Flex>
    );
}

export default ComponentLocationTd;
