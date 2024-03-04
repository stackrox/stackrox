import React from 'react';
import { Flex } from '@patternfly/react-core';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';

export type FixedByVersionProps = {
    fixedByVersion: string;
};

function FixedByVersion({ fixedByVersion }: FixedByVersionProps) {
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

export default FixedByVersion;
