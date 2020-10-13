/* eslint-disable import/prefer-default-export */

export type ScanMessages = {
    header?: string;
    body?: string;
    extra?: string;
};

export const imageScanMessages = {
    missingMetadata: {
        header: 'Unable to retrieve the metadata from the registry',
        body: 'Couldn’t retrieve metadata from the registry, check registry connection.',
        extra: '',
    },
    missingScanData: {
        header: 'There are no scan data available. Please rescan the image.',
        body:
            'Failed to get the base OS information. Either the integrated scanner can’t find the OS or the base OS is unidentifiable.',
        extra: '',
    },
    osUnavailable: {
        header: 'Scanner doesn’t provide OS information',
        body:
            'Failed to get the base OS information. Either the integrated scanner can’t find the OS or the base OS is unidentifiable.',
        extra:
            'Scanner does not provide OS information for any image scanned by DTR, Google, or Anchore; or base image OS doesn’t exists.',
    },
    languageCvesUnavailable: {
        header:
            'Only OS CVE data is available. Unable to retrieve the Language CVE data because it is disabled in a StackRox scanner.',
        body:
            'Only showing information about the OS CVEs. Turn on the Language CVE feature to view additional details.',
        extra: '',
    },
    osCvesUnavailable: {
        header:
            'Only Language CVE data is available. Unable to retrieve the OS CVE data because the scanner doesn’t support this OS.',
        body: 'Unsupported OS. Only Language CVEs are available.',
        extra: '',
    },
    osCvesStale: {
        header: 'The CVE data is no longer being updated.',
        body: 'Stale OS CVE data. The source no longer provides data updates.',
        extra: '',
    },
};
