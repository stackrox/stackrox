import React from 'react';
import { Flex, FlexItem, Label, Popover } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';

import { ScanMessage } from 'messages/vulnMgmt.messages';

export type ImageScanningIncompleteLabelProps = {
    scanMessage: ScanMessage;
};

function ImageScanningIncompleteLabel({ scanMessage }: ImageScanningIncompleteLabelProps) {
    // TODO replace style={{ cursor: 'pointer' }} prop with isClickable prop in PatternFly 6?
    return (
        <Popover
            aria-label="Image scanning incomplete label"
            bodyContent={
                <PopoverBodyContent
                    headerContent="CVE data may be inaccurate"
                    bodyContent={
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                        >
                            <FlexItem>{scanMessage.header}</FlexItem>
                            <FlexItem>{scanMessage.body}</FlexItem>
                        </Flex>
                    }
                />
            }
            enableFlip
            position="top"
        >
            <Label
                color="orange"
                isCompact
                icon={<ExclamationTriangleIcon />}
                variant="outline"
                style={{ cursor: 'pointer' }}
            >
                Image scanning incomplete
            </Label>
        </Popover>
    );
}

export default ImageScanningIncompleteLabel;
