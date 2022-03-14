import rhacsFavicon from 'images/rh-favicon.ico';
import stackroxFavicon from 'images/sr-favicon.ico';
import rhacsLogoSvg from 'images/RHACS-Logo.svg';
import stackroxLogoSvg from 'images/StackRox-Logo.svg';

export type ProductBranding = 'RHACS_BRANDING' | 'STACKROX_BRANDING';

export interface BrandingAssets {
    /** The branding value used to generate assets */
    type: ProductBranding;
    /** The source path to the main branding logo in SVG format */
    logoSvg: string;
    /** Alt text for the main branding logo */
    logoAltText: string;
    /** Value to use as the base in the <title> element */
    basePageTitle: string;
    /** Absolute path to the page favicon */
    favicon: string;
}

const rhacsBranding: BrandingAssets = {
    type: 'RHACS_BRANDING',
    logoSvg: rhacsLogoSvg,
    logoAltText: 'Red Hat Advanced Cluster Security Logo',
    basePageTitle: 'Red Hat Advanced Cluster Security',
    favicon: rhacsFavicon,
};

const stackroxBranding: BrandingAssets = {
    type: 'STACKROX_BRANDING',
    logoSvg: stackroxLogoSvg,
    logoAltText: 'StackRox Logo',
    basePageTitle: 'StackRox',
    favicon: stackroxFavicon,
};

export function getProductBranding(): BrandingAssets {
    const productBranding: string | undefined = process.env.REACT_APP_ROX_PRODUCT_BRANDING;

    switch (productBranding) {
        case 'RHACS_BRANDING':
            return rhacsBranding;
        default:
            return stackroxBranding;
    }
}
