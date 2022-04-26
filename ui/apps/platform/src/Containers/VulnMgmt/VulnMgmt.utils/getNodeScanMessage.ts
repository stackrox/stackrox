import { nodeScanMessages, ScanMessage } from 'messages/vulnMgmt.messages';

export default function getNodeScanMessage(nodeNotes: string[], scanNotes: string[]): ScanMessage {
    const hasMissingScanData = nodeNotes?.includes('MISSING_SCAN_DATA');

    const hasUnsupported = scanNotes?.includes('UNSUPPORTED');
    const hasKernelUnsupported = scanNotes?.includes('KERNEL_UNSUPPORTED');

    if (hasMissingScanData) {
        return nodeScanMessages.missingScanData;
    }
    if (hasUnsupported) {
        return nodeScanMessages.unsupported;
    }
    if (hasKernelUnsupported) {
        return nodeScanMessages.kernelUnsupported;
    }

    return {};
}
