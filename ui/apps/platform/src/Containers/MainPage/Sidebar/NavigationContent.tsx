import React, { CSSProperties, ReactElement, ReactNode } from 'react';
import { Flex, Label } from '@patternfly/react-core';

type NavigationContentVariant = 'Deprecated' | 'TechPreview';

const badgeMap: Record<NavigationContentVariant, ReactElement> = {
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
} as const;

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
