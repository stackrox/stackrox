import React, { ReactElement } from 'react';
import { Brand } from '@patternfly/react-core';

import useBranding from 'hooks/useBranding';

export type BrandLogoProps = {
    className?: string;
};

function BrandLogo(props: BrandLogoProps): ReactElement {
    const branding = useBranding();
    return <Brand {...props} src={branding.logoSvg} alt={branding.logoAltText} />;
}

export default BrandLogo;
