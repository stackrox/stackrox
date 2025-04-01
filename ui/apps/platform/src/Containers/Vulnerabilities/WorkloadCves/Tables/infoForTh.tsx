import React from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';
import { ThProps } from '@patternfly/react-table';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PopoverBodyContent from 'Components/PopoverBodyContent';

export const infoForEpssProbability: ThProps['info'] = {
    // ariaLabel for OutlinedQuestionCircleIcon
    ariaLabel: 'Information about EPSS probability',
    // PopoverBodyContent replaces 5 issues with 1 from axe DevTools:
    // https://dequeuniversity.com/rules/axe/4.10/aria-dialog-name
    // Popover element does not have aria=labelledby attribute
    // rendered if there is a popoverProps.headerContent property.
    popover: (
        <PopoverBodyContent
            headerContent="EPSS probability"
            bodyContent={
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>Likelihood of exploitability</FlexItem>
                    <FlexItem>
                        For more information, see{' '}
                        <ExternalLink>
                            <a
                                href="https://www.first.org/epss/"
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                Exploit Prediction Scoring System (EPSS)
                            </a>
                        </ExternalLink>
                    </FlexItem>
                </Flex>
            }
        />
    ),
};
