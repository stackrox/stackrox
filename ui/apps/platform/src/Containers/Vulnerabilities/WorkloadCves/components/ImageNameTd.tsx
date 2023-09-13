import React from 'react';
import { Flex, Button, ButtonVariant, Truncate } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getEntityPagePath } from '../searchUtils';

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
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <Button
                variant={ButtonVariant.link}
                isInline
                component={LinkShim}
                href={getEntityPagePath('Image', id)}
            >
                <Truncate position="middle" content={`${name.remote}:${name.tag}`} />
            </Button>{' '}
            <span className="pf-u-color-200 pf-u-font-size-sm">in {name.registry}</span>
            <div>{children}</div>
        </Flex>
    );
}

export default ImageNameTd;
