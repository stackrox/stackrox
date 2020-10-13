import { imageScanMessages, ScanMessages } from 'messages/vulnMgmt.messages';

export default function getImageScanMessages(
    imageNotes: string[],
    scanNotes: string[]
): ScanMessages {
    const hasMissingMetadata = imageNotes?.includes('MISSING_METADATA');
    const hasMissingScanData = imageNotes?.includes('MISSING_SCAN_DATA');

    const hasOSUnavailable = scanNotes?.includes('OS_UNAVAILABLE');
    const hasPartialScanData = scanNotes?.includes('PARTIAL_SCAN_DATA');
    const hasLanguageCvesUnavailable = scanNotes?.includes('LANGUAGE_CVES_UNAVAILABLE');
    const hasOSCvesUnavailable = scanNotes?.includes('OS_CVES_UNAVAILABLE');
    const hasOSCvesStale = scanNotes?.includes('OS_CVES_STALE');

    if (hasMissingMetadata) {
        return imageScanMessages.missingMetadata;
    }
    if (hasMissingScanData) {
        return imageScanMessages.missingScanData;
    }
    if (hasOSUnavailable) {
        return imageScanMessages.osUnavailable;
    }
    if (hasPartialScanData && hasLanguageCvesUnavailable) {
        return imageScanMessages.languageCvesUnavailable;
    }
    if (hasPartialScanData && hasOSCvesUnavailable) {
        return imageScanMessages.osCvesUnavailable;
    }
    if (hasOSCvesStale) {
        return imageScanMessages.osCvesStale;
    }

    return {};
}
