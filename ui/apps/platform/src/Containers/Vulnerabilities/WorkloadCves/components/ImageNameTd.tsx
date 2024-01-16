import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { ClipboardCopyButton, Flex, FlexItem, Truncate } from '@patternfly/react-core';

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
            justifyContent={{ default: 'justifyContentSpaceBetween' }}
            alignItems={{ default: 'alignItemsFlexStart' }}
            spaceItems={{ default: 'spaceItemsNone' }}
        >
            <Flex
                grow={{ default: 'grow' }}
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsNone' }}
            >
                <Link to={getEntityPagePath('Image', id, vulnerabilityState)}>
                    <Truncate position="middle" content={`${remote}:${tag}`} />
                </Link>{' '}
                <span className="pf-u-color-200 pf-u-font-size-sm">in {registry}</span>
                <div>{children}</div>
            </Flex>
            <FlexItem shrink={{ default: 'shrink' }}>
                <ClipboardCopyButton
                    id={`copy-image-name-button-${id}`}
                    textId={`copy-image-name-text-${id}`}
                    className="pf-u-pt-xs"
                    variant="plain"
                    exitDelay={1000}
                    onTooltipHidden={() => setCopyIconTooltip('Copy image name')}
                    onClick={copyImageName}
                >
                    {copyIconTooltip}
                </ClipboardCopyButton>
            </FlexItem>
        </Flex>
    );
}

export default ImageNameTd;
