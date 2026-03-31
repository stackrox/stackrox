import type { CSSProperties, ReactElement } from 'react';
import { getProductBranding } from 'constants/productBranding';

export type BrandLogoProps = {
    style?: CSSProperties;
    className?: string;
};

function BrandLogo({ style, className }: BrandLogoProps): ReactElement {
    const branding = getProductBranding();
    const { LogoComponent, logoAltText } = branding;

    return (
        <LogoComponent style={style} className={className} aria-label={logoAltText} role="img" />
    );
}

export default BrandLogo;
