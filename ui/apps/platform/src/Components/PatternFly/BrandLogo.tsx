import React, { ReactElement } from 'react';
import { Brand } from '@patternfly/react-core';
import { getProductBranding } from 'constants/productBranding';

export type BrandLogoProps = {
    className?: string;
};

function BrandLogo(props: BrandLogoProps): ReactElement {
    const branding = getProductBranding();
    return <Brand {...props} src={branding.logoSvg} alt={branding.logoAltText} />;
}

export default BrandLogo;
