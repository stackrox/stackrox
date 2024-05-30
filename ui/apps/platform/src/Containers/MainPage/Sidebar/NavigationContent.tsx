import React, { ReactElement, ReactNode } from 'react';
import { Label } from '@patternfly/react-core';

import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';

type NavigationContentVariant = 'Deprecated' | 'TechPreview';

const badgeMap = {
    Deprecated: (
        <Label
            isCompact
            style={{
                '--pf-v5-c-label__content--Color': 'var(--pf-v5-global--Color--dark-100)',
            }}
        >
            Deprecated
        </Label>
    ),
    TechPreview: <TechPreviewLabel />,
} satisfies Record<string, ReactElement>;

export type NavigationContentProps = {
    children: ReactNode;
    variant: NavigationContentVariant;
};

function NavigationContent({ children, variant }: NavigationContentProps): ReactElement {
    return (
        <>
            <span>{children}</span>
            <span className="pf-v5-u-ml-sm">{badgeMap[variant]}</span>
        </>
    );
}

export default NavigationContent;
