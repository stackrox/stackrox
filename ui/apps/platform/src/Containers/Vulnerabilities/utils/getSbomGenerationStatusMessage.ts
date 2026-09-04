// Image-level notes that indicate the image has no usable scan data at all.
// These genuinely prevent SBOM generation. Scan-level notes such as
// OS_UNAVAILABLE / OS_CVES_UNAVAILABLE / OS_CVES_STALE only affect CVE accuracy
// (the image was still scanned), so they must NOT block SBOM generation — the
// backend endpoint and roxctl produce an SBOM for those images (ROX-36762).
const noScanDataImageNotes = ['MISSING_METADATA', 'MISSING_SCAN_DATA'];

export function hasNoScanData(imageNotes: string[]): boolean {
    return imageNotes?.some((note) => noScanDataImageNotes.includes(note)) ?? false;
}

export function getSbomGenerationStatusMessage({
    isScannerV4Enabled,
    imageNotes,
}: {
    isScannerV4Enabled: boolean;
    imageNotes: string[];
    // scanNotes intentionally does not affect the result; kept for a stable call-site
    // contract where all of an image's notes are passed through.
    scanNotes?: string[];
}): string | undefined {
    if (!isScannerV4Enabled) {
        return 'SBOM generation requires Scanner V4';
    }

    if (hasNoScanData(imageNotes)) {
        return 'SBOM generation is unavailable due to incomplete scan data';
    }

    return undefined;
}
