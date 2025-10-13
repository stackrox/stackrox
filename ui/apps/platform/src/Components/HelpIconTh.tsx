import React from 'react';
import type { ReactElement, ReactNode } from 'react';
import { Button, Flex, FlexItem, Popover } from '@patternfly/react-core';
import { Th } from '@patternfly/react-table';
import type { ThProps } from '@patternfly/react-table';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

type HelpIconThProps = {
    children: string | ReactNode;
    sort?: ThProps['sort'];
    popoverContent: ReactElement;
};

function HelpIconTh({ children, sort, popoverContent }: HelpIconThProps) {
    return (
        <Th sort={sort || undefined}>
            <Flex
                direction={{ default: 'row' }}
                alignItems={{ default: 'alignItemsCenter' }}
                flexWrap={{ default: 'nowrap' }}
            >
                <FlexItem>{children}</FlexItem>
                <FlexItem>
                    <Popover aria-label="Table column info" bodyContent={popoverContent}>
                        <Button
                            variant="plain"
                            isInline
                            aria-label="Show table column info"
                            className="pf-v5-u-p-0"
                        >
                            <OutlinedQuestionCircleIcon />
                        </Button>
                    </Popover>
                </FlexItem>
            </Flex>
        </Th>
    );
}

export default HelpIconTh;
