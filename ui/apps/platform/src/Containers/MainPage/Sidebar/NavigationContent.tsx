import React, { CSSProperties, ReactElement, ReactNode } from 'react';
import { Flex, Label } from '@patternfly/react-core';

const badgeMap = {
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
    TechPreview: (
        <Label isCompact color="orange">
            Tech preview
        </Label>
    ),
} satisfies Record<string, ReactElement>;

type NavigationContentVariant = keyof typeof badgeMap;

export type NavigationContentProps = {
    children: ReactNode;
    variant: NavigationContentVariant;
};

function NavigationContent({ children, variant }: NavigationContentProps): ReactElement {
    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            <span>{children}</span>
            <span>{badgeMap[variant]}</span>
        </Flex>
    );
}

export default NavigationContent;
