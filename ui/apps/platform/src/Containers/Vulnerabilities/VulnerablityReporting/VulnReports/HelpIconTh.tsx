import React from 'react';
import { Flex, FlexItem, Tooltip } from '@patternfly/react-core';
import { Th, ThProps } from '@patternfly/react-table';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

type TooltipThProps = {
    children: string | React.ReactNode;
    sort?: ThProps['sort'];
    tooltip: string;
};

function HelpIconTh({ children, tooltip, sort }: TooltipThProps) {
    return (
        <Th sort={sort || undefined}>
            <Flex direction={{ default: 'row' }} alignItems={{ default: 'alignItemsCenter' }}>
                <FlexItem>{children}</FlexItem>
                <FlexItem>
                    <Tooltip content={tooltip}>
                        <OutlinedQuestionCircleIcon />
                    </Tooltip>
                </FlexItem>
            </Flex>
        </Th>
    );
}

export default HelpIconTh;
