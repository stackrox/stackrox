import React from 'react';
import { Button, Flex, FlexItem, Label, Popover } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';

import { ScanMessage } from 'messages/vulnMgmt.messages';

export type ImageScanningIncompleteLabelProps = {
    scanMessage: ScanMessage;
};

function ImageScanningIncompleteLabel({ scanMessage }: ImageScanningIncompleteLabelProps) {
    return (
        <>
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
                <Button variant="plain" className="pf-v5-u-p-0">
                    <Label
                        color="orange"
                        isCompact
                        icon={<ExclamationTriangleIcon />}
                        variant="outline"
                    >
                        Image scanning incomplete
                    </Label>
                </Button>
            </Popover>
        </>
    );
}

export default ImageScanningIncompleteLabel;
