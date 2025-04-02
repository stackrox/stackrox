import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { Button, Flex, FlexItem, Tooltip, Truncate } from '@patternfly/react-core';
import { OutlinedCopyIcon } from '@patternfly/react-icons';

import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import { getImageBaseNameDisplay } from '../utils/images';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import useClipboardCopy from 'hooks/useClipboardCopy';

export type ImageNameLinkProps = {
    name: {
        remote: string;
        registry: string;
        tag: string;
    };
    id: string;
    children?: React.ReactNode;
};

function ImageNameLink({ name, id, children }: ImageNameLinkProps) {
    const { getAbsoluteUrl } = useWorkloadCveViewContext();
    const vulnerabilityState = useVulnerabilityState();
    const [copyIconTooltip, setCopyIconTooltip] = useState('Copy image name');
    const { copyToClipboard } = useClipboardCopy();

    const { registry } = name;

    // If tag is not provided, use the image hash (id) full the full image name
    const baseName = getImageBaseNameDisplay(id, name);

    function copyImageName() {
        copyToClipboard(`${registry}/${baseName}`).then(() => setCopyIconTooltip('Copied!'));
    }

    return (
        <Flex
            direction={{ default: 'row' }}
            flexWrap={{ default: 'nowrap' }}
            alignItems={{ default: 'alignItemsFlexStart' }}
            spaceItems={{ default: 'spaceItemsNone' }}
        >
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                <Link
                    to={getAbsoluteUrl(getWorkloadEntityPagePath('Image', id, vulnerabilityState))}
                >
                    <Truncate position="middle" content={baseName} />
                </Link>{' '}
                <span className="pf-v5-u-color-200 pf-v5-u-font-size-sm">in {registry}</span>
                <div>{children}</div>
            </Flex>
            <FlexItem>
                <Tooltip
                    trigger="mouseenter focus click"
                    aria-live="polite"
                    aria="none"
                    exitDelay={1000}
                    onTooltipHidden={() => setCopyIconTooltip('Copy image name')}
                    content={<div>{copyIconTooltip}</div>}
                >
                    <Button
                        className="pf-v5-u-pt-xs"
                        id={`copy-image-name-button-${id}`}
                        aria-label={'Copy image name'}
                        type="button"
                        variant="plain"
                        onClick={copyImageName}
                    >
                        <OutlinedCopyIcon />
                    </Button>
                </Tooltip>
            </FlexItem>
        </Flex>
    );
}

export default ImageNameLink;
