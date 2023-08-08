import React, { ReactElement } from 'react';
import { Button, Flex, FlexItem, Popover } from '@patternfly/react-core';
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
                        <Button variant="plain" aria-label="Show table column info">
                            <OutlinedQuestionCircleIcon />
                        </Button>
                    </Popover>
                </FlexItem>
            </Flex>
        </Th>
    );
}

export default HelpIconTh;
