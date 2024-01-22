import React from 'react';
import { Button, Flex, FlexItem, Label, Popover } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';
import isEmpty from 'lodash/isEmpty';

import getImageScanMessage from '../utils/getImageScanMessage';

export type ImageScanningErrorLabelProps = {
    imageNotes: string[];
    scanNotes: string[];
};

function ImageScanningErrorLabel({ imageNotes, scanNotes }: ImageScanningErrorLabelProps) {
    const scanMessage = getImageScanMessage(imageNotes, scanNotes);

    if (isEmpty(scanMessage)) {
        return null;
    }

    return (
        <>
            <Popover
                aria-label="Image scanning error label"
                headerContent={<div>CVE data may be inaccurate</div>}
                bodyContent={
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <FlexItem>{scanMessage.header}</FlexItem>
                        <FlexItem>{scanMessage.body}</FlexItem>
                    </Flex>
                }
                enableFlip
                position="top"
            >
                <Button variant="plain" className="pf-u-p-0">
                    <Label
                        color="orange"
                        isCompact
                        icon={<ExclamationTriangleIcon />}
                        variant="outline"
                    >
                        Image scanning error
                    </Label>
                </Button>
            </Popover>
        </>
    );
}

export default ImageScanningErrorLabel;
