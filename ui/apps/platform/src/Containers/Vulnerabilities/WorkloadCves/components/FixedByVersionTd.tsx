import React from 'react';
import { Flex } from '@patternfly/react-core';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';

export type FixedByVersionTdProps = {
    fixedByVersion: string;
};

function FixedByVersionTd({ fixedByVersion }: FixedByVersionTdProps) {
    return fixedByVersion !== '' ? (
        <>{fixedByVersion}</>
    ) : (
        <Flex
            alignItems={{ default: 'alignItemsCenter' }}
            spaceItems={{ default: 'spaceItemsSm' }}
            flexWrap={{ default: 'nowrap' }}
        >
            <VulnerabilityFixableIconText isFixable={false} />
        </Flex>
    );
}

export default FixedByVersionTd;
