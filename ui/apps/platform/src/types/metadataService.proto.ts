export type Metadata = {
    version: string;
    buildFlavor: string;
    releaseBuild: boolean;
    licenseStatus: string;
    versionString?: string;
    productBranding: ProductBranding;
};

export type ProductBranding = 'RHACS_BRANDING' | 'STACKROX_BRANDING';
