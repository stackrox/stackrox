import { nodeScanMessages } from 'messages/vulnMgmt.messages';
import getNodeScanMessage from './getNodeScanMessage';

describe('getNodeScanMessage', () => {
    it('should return an empty object when there are no notes in the notes arrays', () => {
        const nodesNotes = [];
        const scanNotes = [];

        const messageObj = getNodeScanMessage(nodesNotes, scanNotes);

        expect(messageObj).toEqual({});
    });

    it('should return an object for missingScanData when node notes contain MISSING_SCAN_DATA', () => {
        const nodesNotes = ['MISSING_SCAN_DATA'];
        const scanNotes = [];

        const messageObj = getNodeScanMessage(nodesNotes, scanNotes);

        expect(messageObj).toEqual(nodeScanMessages.missingScanData);
    });

    it('should return an object for osUnsupported when scan notes contain UNSUPPORTED', () => {
        const nodesNotes = [];
        const scanNotes = ['UNSUPPORTED'];

        const messageObj = getNodeScanMessage(nodesNotes, scanNotes);

        expect(messageObj).toEqual(nodeScanMessages.unsupported);
    });

    it('should return an object for kernelUnsupported when scan notes contain KERNEL_UNSUPPORTED', () => {
        const nodesNotes = [];
        const scanNotes = ['KERNEL_UNSUPPORTED'];

        const messageObj = getNodeScanMessage(nodesNotes, scanNotes);

        expect(messageObj).toEqual(nodeScanMessages.kernelUnsupported);
    });

    it('should return an object for certifiedRHELCVEsUnavailable when scan notes contain CERTIFIED_RHEL_CVES_UNAVAILABLE', () => {
        const nodesNotes = [];
        const scanNotes = ['CERTIFIED_RHEL_CVES_UNAVAILABLE'];

        const messageObj = getNodeScanMessage(nodesNotes, scanNotes);

        expect(messageObj).toEqual(nodeScanMessages.certifiedRHELCVEsUnavailable);
    });
});
