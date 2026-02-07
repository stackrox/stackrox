import type { ReactElement, ReactNode } from 'react';
import { Label } from '@patternfly/react-core';

import TechPreviewLabel from 'Components/PatternFly/PreviewLabel/TechPreviewLabel';

type NavigationContentVariant = 'Deprecated' | 'TechPreview';

const badgeMap = {
    Deprecated: <Label isCompact>Deprecated</Label>,
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
            <span className="pf-v6-u-ml-sm">{badgeMap[variant]}</span>
        </>
    );
}

export default NavigationContent;
