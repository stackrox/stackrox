/* eslint-disable import/no-duplicates */
import type { ComponentType, SVGProps } from 'react';
import rhacsFavicon from 'images/rh-favicon.ico';
import stackroxFavicon from 'images/sr-favicon.ico';
import rhacsLogoSvg from 'images/RHACS-Logo.svg';
import RHACSLogoSvg from 'images/RHACS-Logo.svg?react';
import stackroxLogoSvg from 'images/StackRox-Logo.svg';
import StackRoxLogoSvg from 'images/StackRox-Logo.svg?react';
/* eslint-enable import/no-duplicates */

export type ProductBranding = 'RHACS_BRANDING' | 'STACKROX_BRANDING';

export interface BrandingAssets {
    /** The branding value used to generate assets */
    type: ProductBranding;
    /** The source path to the main branding logo in SVG format */
    /** Retained only for PDF generation */
    logoSvg: string;
    /** React component for the logo SVG */
    LogoComponent: ComponentType<SVGProps<SVGSVGElement>>;
    /** Alt text for the main branding logo */
    logoAltText: string;
    /** Value to use as the base in the <title> element */
    basePageTitle: string;
    /** Value for default subject of report e-mail */
    reportName: string;
    /** Shortened version of product name */
    shortName: string;
    /** Absolute path to the page favicon */
    favicon: string;
}

const rhacsBranding: BrandingAssets = {
    type: 'RHACS_BRANDING',
    logoSvg: rhacsLogoSvg,
    LogoComponent: RHACSLogoSvg,
    logoAltText: 'Red Hat Advanced Cluster Security Logo',
    basePageTitle: 'Red Hat Advanced Cluster Security',
    reportName: 'Red Hat Advanced Cluster Security (RHACS)',
    shortName: 'RHACS',
    favicon: rhacsFavicon,
};

const stackroxBranding: BrandingAssets = {
    type: 'STACKROX_BRANDING',
    logoSvg: stackroxLogoSvg,
    LogoComponent: StackRoxLogoSvg,
    logoAltText: 'StackRox Logo',
    basePageTitle: 'StackRox',
    reportName: 'StackRox',
    shortName: 'StackRox',
    favicon: stackroxFavicon,
};

// @TODO: This should be renamed to getProductBrandingAssets to be more specific. It would be nice
// to have a function to just get the product brand itself (ie. RHACS_BRANDING, STACKROX_BRANDING)
export function getProductBranding(): BrandingAssets {
    const productBranding: string | undefined = process.env.ROX_PRODUCT_BRANDING;

    switch (productBranding) {
        case 'RHACS_BRANDING':
            return rhacsBranding;
        default:
            return stackroxBranding;
    }
}
