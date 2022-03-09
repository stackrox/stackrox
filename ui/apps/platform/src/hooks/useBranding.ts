import React from 'react';
import { useSelector } from 'react-redux';
import { createSelector } from 'reselect';
import { selectors } from 'reducers';

import { ProductBranding, Metadata } from 'types/metadataService.proto';
import rhacsFavicon from 'images/rh-favicon.ico';
import stackroxFavicon from 'images/sr-favicon.ico';
import rhacsLogoSvg from 'images/RHACS-Logo.svg';
import stackroxLogoSvg from 'images/StackRox-Logo.svg';

const selectMetadata = createSelector([selectors.getMetadata], (metadata: Metadata) => metadata);

export interface BrandingAssets {
    /** The branding value returned from Central used to generate assets */
    type: ProductBranding | null;
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

// Empty asset values to prevent incorrect branding in the case of a logic change that
// would cause the default case to fire.
const fallbackAssets: BrandingAssets = {
    type: null,
    logoSvg: '',
    logoAltText: '',
    basePageTitle: '',
    favicon: '',
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
                `An invalid value for 'productBranding' was returned from Central, page assets may be missing.`
            );
            return fallbackAssets;
    }
}

/**
 * Wrapper to create a HOC that allows usage of the `useBranding` logic in class components.
 * Note that any `branding` prop passed to this wrapped component will be overwritten.
 *
 * @param Component A React component that has a required `branding` prop
 * @returns A component with the `branding` prop provided by the `useBranding` hook
 */
export function withBranding<Props extends { branding: BrandingAssets }>(
    Component: React.ComponentType<Props>
) {
    return function HOCBranded(props: Omit<Props, 'branding'>) {
        const branding = useBranding();
        return React.createElement(Component, { ...(props as Props), branding }, null);
    };
}

export default useBranding;
