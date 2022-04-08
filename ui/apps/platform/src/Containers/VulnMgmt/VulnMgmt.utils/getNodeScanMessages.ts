import { nodeScanMessages, ScanMessages } from 'messages/vulnMgmt.messages';

export default function getNodeScanMessages(
    nodeNotes: string[],
    scanNotes: string[]
): ScanMessages {
    const hasMissingScanData = nodeNotes?.includes('MISSING_SCAN_DATA');

    const hasOSUnsupported = scanNotes?.includes('OS_UNSUPPORTED');

    if (hasMissingScanData) {
        return nodeScanMessages.missingScanData;
    }
    if (hasOSUnsupported) {
        return nodeScanMessages.osUnsupported;
    }

    return {};
}
