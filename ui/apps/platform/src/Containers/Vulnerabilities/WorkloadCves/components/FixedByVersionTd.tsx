import React from 'react';
import { Flex } from '@patternfly/react-core';
import { NotFixableIcon } from 'Components/PatternFly/FixabilityIcons';

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
            <div className="pf-u-flex-basis-auto">
                <NotFixableIcon />
            </div>
            <span>Not fixable</span>
        </Flex>
    );
}

export default FixedByVersionTd;
