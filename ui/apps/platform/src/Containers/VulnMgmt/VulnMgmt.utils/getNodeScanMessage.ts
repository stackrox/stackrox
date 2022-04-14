import { nodeScanMessages, ScanMessage } from 'messages/vulnMgmt.messages';

export default function getNodeScanMessage(nodeNotes: string[], scanNotes: string[]): ScanMessage {
    const hasMissingScanData = nodeNotes?.includes('MISSING_SCAN_DATA');

    const hasOSUnsupported = scanNotes?.includes('OS_UNSUPPORTED');
    const hasKernelUnsupported = scanNotes?.includes('KERNEL_UNSUPPORTED');

    if (hasMissingScanData) {
        return nodeScanMessages.missingScanData;
    }
    if (hasOSUnsupported) {
        return nodeScanMessages.osUnsupported;
    }
    if (hasKernelUnsupported) {
        return nodeScanMessages.kernelUnsupported;
    }

    return {};
}
