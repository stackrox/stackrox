import React from 'react';
import { Flex, Button, ButtonVariant } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getEntityPagePath } from '../searchUtils';

type ImageNameTdProps = {
    name: {
        remote: string;
        registry: string;
    };
    id: string;
};

function ImageNameTd({ name, id }: ImageNameTdProps) {
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <Button
                variant={ButtonVariant.link}
                isInline
                component={LinkShim}
                href={getEntityPagePath('Image', id)}
            >
                {name.remote}
            </Button>{' '}
            <span className="pf-u-color-200 pf-u-font-size-sm">in {name.registry}</span>
        </Flex>
    );
}

export default ImageNameTd;
