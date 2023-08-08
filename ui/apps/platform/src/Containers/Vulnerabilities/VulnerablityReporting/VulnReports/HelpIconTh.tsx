import React, { ReactElement } from 'react';
import { Flex, FlexItem, Popover } from '@patternfly/react-core';
import { Th, ThProps } from '@patternfly/react-table';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

type HelpIconThProps = {
    children: string | React.ReactNode;
    sort?: ThProps['sort'];
    popoverContent: ReactElement;
};

function HelpIconTh({ children, sort, popoverContent }: HelpIconThProps) {
    return (
        <Th sort={sort || undefined}>
            <Flex direction={{ default: 'row' }} alignItems={{ default: 'alignItemsCenter' }}>
                <FlexItem>{children}</FlexItem>
                <FlexItem>
                    <Popover aria-label="Table column info" bodyContent={popoverContent}>
                        <OutlinedQuestionCircleIcon aria-labelledby="Show table column info" />
                    </Popover>
                </FlexItem>
            </Flex>
        </Th>
    );
}

export default HelpIconTh;
