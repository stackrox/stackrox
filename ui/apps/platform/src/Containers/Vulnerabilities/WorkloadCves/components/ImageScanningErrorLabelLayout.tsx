import React from 'react';
import { Button, Flex, FlexItem, Label, Popover } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';
import isEmpty from 'lodash/isEmpty';

import getImageScanMessage from '../utils/getImageScanMessage';

export type ImageScanningErrorLabelLayoutProps = {
    children: React.ReactNode;
    imageNotes: string[];
    scanNotes: string[];
};

/**
 * ‘Image scanning error’ label layout for use in tables. Conditionally renders a label
 * with a tooltip for information about image scanning errors.
 *
 * @param children - The table cell contents to render before the label
 * @param imageNotes - The image notes
 * @param scanNotes - The image scan notes
 */
function ImageScanningErrorLabelLayout({
    children,
    imageNotes,
    scanNotes,
}: ImageScanningErrorLabelLayoutProps) {
    const scanMessage = getImageScanMessage(imageNotes, scanNotes);

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
            {children}
            {!isEmpty(scanMessage) && (
                <FlexItem>
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
                </FlexItem>
            )}
        </Flex>
    );
}

export default ImageScanningErrorLabelLayout;
