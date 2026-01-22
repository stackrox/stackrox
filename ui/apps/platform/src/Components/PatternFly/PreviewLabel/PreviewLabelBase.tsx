import { Flex, FlexItem, Label, Popover } from '@patternfly/react-core';
import type { LabelProps } from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PopoverBodyContent from 'Components/PopoverBodyContent';

type PreviewLabelBaseProps = {
    ariaLabel: string;
    title: string;
    body: string;
    color: LabelProps['color'];
};

function PreviewLabelBase({ ariaLabel, title, body, color }: PreviewLabelBaseProps) {
    return (
        <Popover
            aria-label={ariaLabel}
            bodyContent={
                <PopoverBodyContent
                    headerContent={title}
                    bodyContent={
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsXs' }}
                        >
                            <FlexItem>{body}</FlexItem>
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
            <Label isCompact color={color} icon={<InfoCircleIcon />} style={{ cursor: 'pointer' }}>
                {title}
            </Label>
        </Popover>
    );
}

export default PreviewLabelBase;
