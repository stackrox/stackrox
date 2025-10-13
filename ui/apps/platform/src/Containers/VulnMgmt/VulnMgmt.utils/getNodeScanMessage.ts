import { nodeScanMessages, ScanMessage } from 'messages/vulnMgmt.messages';

export default function getNodeScanMessage(nodeNotes: string[], scanNotes: string[]): ScanMessage {
    const hasMissingScanData = nodeNotes?.includes('MISSING_SCAN_DATA');

    const hasUnsupported = scanNotes?.includes('UNSUPPORTED');
    const hasKernelUnsupported = scanNotes?.includes('KERNEL_UNSUPPORTED');
    const hasCertifiedRHELCVEsUnavailable = scanNotes?.includes('CERTIFIED_RHEL_CVES_UNAVAILABLE');

    if (hasMissingScanData) {
        return nodeScanMessages.missingScanData;
    }
    if (hasUnsupported) {
        return nodeScanMessages.unsupported;
    }
    if (hasKernelUnsupported) {
        return nodeScanMessages.kernelUnsupported;
    }
    if (hasCertifiedRHELCVEsUnavailable) {
        return nodeScanMessages.certifiedRHELCVEsUnavailable;
    }

    return {};
}
