import React, { CSSProperties, ReactElement, ReactNode } from 'react';
import { Label } from '@patternfly/react-core';

import TechPreviewLabel from 'Components/PatternFly/TechPreviewLabel';

type NavigationContentVariant = 'Deprecated' | 'TechPreview';

const style = {
    '--pf-v5-c-label__content--Color': 'var(--pf-v5-global--Color--dark-100)',
} as CSSProperties;
// Type assertion prevents TypeScript error:
// error TS2353: Object literal may only specify known properties, and ''--pf-v5-c-label__content--Color'' does not exist in type 'Properties<string | number, string & {}>'.
// The expected type comes from property 'style' which is declared here on type 'IntrinsicAttributes & LabelProps'

const badgeMap = {
    Deprecated: (
        <Label isCompact style={style}>
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
