import React from 'react';
import { Flex, Tooltip, Truncate } from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

import { SourceType } from '../Tables/table.utils';

export type ComponentLocationProps = {
    location: string;
    source: SourceType;
};

function ComponentLocation({ location, source }: ComponentLocationProps) {
    return (
        <Flex spaceItems={{ default: 'spaceItemsXs' }} alignItems={{ default: 'alignItemsCenter' }}>
            <Truncate content={location || 'N/A'} position="middle" />
            {source === 'OS' && location === '' && (
                <Tooltip content="Location is unavailable for operating system packages">
                    <InfoCircleIcon color="var(--pf-global--info-color--100)" />
                </Tooltip>
            )}
        </Flex>
    );
}

export default ComponentLocation;
