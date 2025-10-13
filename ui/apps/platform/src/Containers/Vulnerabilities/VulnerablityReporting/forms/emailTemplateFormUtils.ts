import { getProductBranding } from 'constants/productBranding';

// Helper functions

const { type: productBrand } = getProductBranding();
const productBrandText =
    productBrand === 'RHACS_BRANDING' ? 'Red Hat Advanced Cluster Security (RHACS)' : 'StackRox';
const shortName = productBrand === 'RHACS_BRANDING' ? 'RHACS' : 'StackRox';

export const defaultEmailSubjectTemplate = `${shortName} Workload CVE Report for <Config name>; Scope: <Collection name>`;

export const defaultEmailBody = `${productBrandText} for Kubernetes has identified workload CVEs in the images matched by the following report configuration parameters. The attached Vulnerability report lists those workload CVEs and associated details to help with remediation. Please review the vulnerable software packages/components from the impacted images and update them to a version containing the fix, if one is available.\n\nPlease note that an attachment of the report data will not be provided if no CVEs are found.`;

export const defaultEmailBodyWithNoCVEsFound = `${productBrandText} for Kubernetes has identified no workload CVEs in the images matched by the following report configuration parameters.\n\nPlease note that an attachment of the report data will not be provided if no CVEs are found.`;

export function getDefaultEmailSubject(reportName, reportScope = ''): string {
    return defaultEmailSubjectTemplate
        .replace('<Config name>', reportName)
        .replace('<Collection name>', reportScope);
}
