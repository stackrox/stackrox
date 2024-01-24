import isEmpty from 'lodash/isEmpty';
import Raven from 'raven-js';

import { imageScanMessages, ScanMessage } from 'messages/vulnMgmt.messages';

export default function getImageScanMessage(
    imageNotes: string[],
    scanNotes: string[]
): ScanMessage {
    const hasMissingMetadata = imageNotes?.includes('MISSING_METADATA');
    const hasMissingScanData = imageNotes?.includes('MISSING_SCAN_DATA');

    const hasOSUnavailable = scanNotes?.includes('OS_UNAVAILABLE');
    const hasPartialScanData = scanNotes?.includes('PARTIAL_SCAN_DATA');
    const hasLanguageCvesUnavailable = scanNotes?.includes('LANGUAGE_CVES_UNAVAILABLE');
    const hasOSCvesUnavailable = scanNotes?.includes('OS_CVES_UNAVAILABLE');
    const hasOSCvesStale = scanNotes?.includes('OS_CVES_STALE');
    const hasCertifiedRHELCvesUnavailable = scanNotes?.includes('CERTIFIED_RHEL_SCAN_UNAVAILABLE');

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
    if (hasPartialScanData && hasCertifiedRHELCvesUnavailable) {
        return imageScanMessages.certifiedRHELUnavailable;
    }
    if (hasOSCvesStale) {
        return imageScanMessages.osCvesStale;
    }
    if (!isEmpty(imageNotes) || !isEmpty(scanNotes)) {
        Raven.captureException(new Error('Unknown State Detected'), {
            extra: { imageNotes, scanNotes },
        });
    }

    return {};
}
