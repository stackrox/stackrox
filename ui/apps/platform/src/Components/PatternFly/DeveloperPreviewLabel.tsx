import React from 'react';
import { Flex, FlexItem, Label, Popover } from '@patternfly/react-core';
import type { LabelProps } from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PopoverBodyContent from 'Components/PopoverBodyContent';

export type DeveloperPreviewLabelProps = Omit<LabelProps, 'children'>;

function DeveloperPreviewLabel({ className, ...props }: DeveloperPreviewLabelProps) {
    return (
        <Popover
            aria-label="Developer preview info"
            bodyContent={
                <PopoverBodyContent
                    headerContent="Developer preview"
                    bodyContent={
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsXs' }}
                        >
                            <FlexItem>
                                Developer preview features are not intended to be used in production
                                environments. The clusters deployed with the developer preview
                                features are considered to be development cluster and are not
                                supported through the Red Hat Customer Portal case management
                                system.
                            </FlexItem>
                            <FlexItem>
                                <ExternalLink>
                                    <a
                                        href="https://access.redhat.com/articles/6966848"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        onClick={(e) => e.stopPropagation()}
                                    >
                                        Learn more
                                    </a>
                                </ExternalLink>
                            </FlexItem>
                        </Flex>
                    }
                />
            }
            enableFlip
            position="top"
        >
            <Label
                isCompact
                color="purple"
                icon={<InfoCircleIcon />}
                className={className}
                style={{ cursor: 'pointer' }}
                {...props}
            >
                Developer preview
            </Label>
        </Popover>
    );
}

export default DeveloperPreviewLabel;
