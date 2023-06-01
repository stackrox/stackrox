import React, { CSSProperties } from 'react';
import { Flex, Label } from '@patternfly/react-core';

import LeftNavItem, { LeftNavItemProps } from './LeftNavItem';

export type BadgedNavItemProps = LeftNavItemProps & {
    variant: 'TechPreview' | 'Deprecated';
};

const badges = {
    TechPreview: (
        <Label isCompact color="orange">
            Tech preview
        </Label>
    ),
    Deprecated: (
        <Label
            isCompact
            style={
                {
                    '--pf-c-label--BackgroundColor': 'var(--pf-global--disabled-color--200)',
                } as CSSProperties
            }
        >
            Deprecated
        </Label>
    ),
} as const;

function BadgedNavItem(props: BadgedNavItemProps) {
    return (
        <LeftNavItem
            {...props}
            title={
                <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                    <span>{props.title}</span>
                    <span>{badges[props.variant]}</span>
                </Flex>
            }
        />
    );
}

export default BadgedNavItem;
