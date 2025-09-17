import React from 'react';
import { Button, Flex, FlexItem, Label, LabelProps, Popover } from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';

export type DeveloperPreviewLabelProps = Omit<LabelProps, 'children'>;

function DeveloperPreviewLabel({ className, ...props }: DeveloperPreviewLabelProps) {
    const popoverContent = (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
            <FlexItem>
                Developer preview features are not intended to be used in production environments.
                The clusters deployed with the developer preview features are considered to be
                development cluster and are not supported through the Red Hat Customer Portal case
                management system.
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
    );

    return (
        <Label
            isCompact
            color="purple"
            className={`pf-v5-u-font-weight-light pf-v5-u-font-family-sans-serif ${className ?? ''}`}
            {...props}
        >
            <Flex
                direction={{ default: 'row' }}
                alignItems={{ default: 'alignItemsCenter' }}
                spaceItems={{ default: 'spaceItemsXs' }}
            >
                <FlexItem>
                    <Popover
                        aria-label="Developer preview info"
                        bodyContent={popoverContent}
                        position="top"
                        triggerAction="hover"
                    >
                        <Button
                            variant="plain"
                            color="purple"
                            isInline
                            aria-label="Show developer preview info"
                            className="pf-v5-u-p-0 pf-v5-u-color-purple-500"
                        >
                            <InfoCircleIcon color="var(--pf-v5-global--palette--purple-500)" />
                        </Button>
                    </Popover>
                </FlexItem>
                <FlexItem>Developer preview</FlexItem>
            </Flex>
        </Label>
    );
}

export default DeveloperPreviewLabel;
