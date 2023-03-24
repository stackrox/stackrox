export type ScanMessage = {
    header?: string;
    body?: string;
};

export const imageScanMessages = {
    missingMetadata: {
        header: 'Failed to retrieve metadata from the registry.',
        body: 'Couldn’t retrieve metadata from the registry, check registry connection.',
    },
    missingScanData: {
        header: 'Failed to get the base OS information.',
        body: 'Failed to get the base OS information. Either the integrated scanner can’t find the OS or the base OS is unidentifiable.',
    },
    osUnavailable: {
        header: 'The scanner doesn’t provide OS information.',
        body: 'Failed to get the base OS information. Either the integrated scanner can’t find the OS or the base OS is unidentifiable.',
    },
    languageCvesUnavailable: {
        header: 'Unable to retrieve the Language CVE data, only OS CVE data is available.',
        body: 'Only showing information about the OS CVEs. Turn on the Language CVE feature in a scanner to view additional details.',
    },
    osCvesUnavailable: {
        header: 'Unable to retrieve the OS CVE data, only Language CVE data is available.',
        body: 'Only showing information about the Language CVEs.',
    },
    osCvesStale: {
        header: 'Stale OS CVE data.',
        body: 'The source no longer provides data updates.',
        extra: '',
    },
    certifiedRHELUnavailable: {
        header: 'Image out of scope for Red Hat Vulnerability Scanner Certification.',
        body: 'The scan results are not certified, as the base RHEL image is out of scope for certification. Please consider updating the base image.',
    },
};

export const nodeScanMessages = {
    missingScanData: {
        header: 'Failed to get scan data.',
        body: 'Failed to get scan data. There may have been an error communicating with the integrated node scanner.',
    },
    unsupported: {
        header: 'Node unsupported.',
        body: 'Scanning this node is not supported at this time. Please see the release notes for more information.',
    },
    kernelUnsupported: {
        header: 'Node’s kernel unsupported.',
        body: 'This node’s kernel is not supported at this time.',
    },
    certifiedRHELCVEsUnavailable: {
        header: 'Unable to scan node components',
        body: 'The list of packages is missing scan results because the data required to conduct a scan is unavailable for this node.',
    },
};
