import { useSelector } from 'react-redux';
import { createSelector } from 'reselect';
import { selectors } from 'reducers';

import { ProductBranding, Metadata } from 'types/metadataService.proto';
import rhacsFavicon from 'images/rh-favicon.ico';
import stackroxFavicon from 'images/sr-favicon.ico';
import rhacsLogoSvg from 'images/RHACS-Logo.svg';
import stackroxLogoSvg from 'images/StackRox-Logo.svg';
import rhacsLogoPng from 'images/RHACS-Logo.png';
import stackroxLogoPng from 'images/stackrox-logo.png';

const selectMetadata = createSelector([selectors.getMetadata], (metadata: Metadata) => metadata);

export interface BrandingAssets {
    /** The branding value returned from Central used to generate assets */
    type: ProductBranding;
    /** The source path to the main branding logo in SVG format */
    logoSvg: string;
    /** The source path to the main branding logo in PNG format */
    logoPng: string;
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
    logoPng: rhacsLogoPng,
    logoAltText: 'Red Hat Advanced Cluster Security',
    basePageTitle: 'Red Hat Advanced Cluster Security',
    favicon: rhacsFavicon,
};

const stackroxBranding: BrandingAssets = {
    type: 'STACKROX_BRANDING',
    logoSvg: stackroxLogoSvg,
    logoPng: stackroxLogoPng,
    logoAltText: 'StackRox',
    basePageTitle: 'StackRox',
    favicon: stackroxFavicon,
};

function useBranding(): BrandingAssets {
    const { productBranding }: Metadata = useSelector(selectMetadata);

    switch (productBranding) {
        case 'RHACS_BRANDING':
            return rhacsBranding;
        case 'STACKROX_BRANDING':
            return stackroxBranding;
        default:
            // eslint-disable-next-line no-console
            console.warn(
                `An invalid value for 'productBranding' was returned from Central, defaulting to open source branding.`
            );
            return stackroxBranding;
    }
}

export default useBranding;
