import React, { CSSProperties, ReactElement, ReactNode } from 'react';
import { Flex, Label } from '@patternfly/react-core';

import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';

type NavigationContentVariant = 'Deprecated' | 'TechPreview';

const badgeMap: Record<string, ReactElement> = {
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
    TechPreview: <TechPreviewLabel />,
}; // TODO why does tsc build fail with missing semicolon with satisfies Record<string, ReactElement>

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
