import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { Button, Flex, FlexItem, Tooltip, Truncate } from '@patternfly/react-core';
import { OutlinedCopyIcon } from '@patternfly/react-icons';

import { getEntityPagePath } from '../searchUtils';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

export type ImageNameTdProps = {
    name: {
        remote: string;
        registry: string;
        tag: string;
    };
    id: string;
    children?: React.ReactNode;
};

function ImageNameTd({ name, id, children }: ImageNameTdProps) {
    const vulnerabilityState = useVulnerabilityState();
    const [copyIconTooltip, setCopyIconTooltip] = useState('Copy image name');

    const { registry, remote, tag } = name;

    function copyImageName() {
        navigator?.clipboard
            ?.writeText(`${registry}/${remote}:${tag}`)
            .then(() => setCopyIconTooltip('Copied!'))
            .catch(() => {}); /* Nothing to do */
    }

    return (
        <Flex
            direction={{ default: 'row' }}
            flexWrap={{ default: 'nowrap' }}
            alignItems={{ default: 'alignItemsFlexStart' }}
            spaceItems={{ default: 'spaceItemsNone' }}
        >
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
                <Link to={getEntityPagePath('Image', id, vulnerabilityState)}>
                    <Truncate position="middle" content={`${remote}:${tag}`} />
                </Link>{' '}
                <span className="pf-u-color-200 pf-u-font-size-sm">in {registry}</span>
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
                        className="pf-u-pt-xs"
                        id={`copy-image-name-button-${id}`}
                        aria-labelledby={`copy-image-name-text-${id}`}
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

export default ImageNameTd;
