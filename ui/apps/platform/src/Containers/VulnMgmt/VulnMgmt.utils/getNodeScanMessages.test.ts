import { nodeScanMessages } from 'messages/vulnMgmt.messages';
import getNodeScanMessages from './getNodeScanMessages';

describe('getNodeScanMessages', () => {
    it('should return an empty object when there are no notes in the notes arrays', () => {
        const nodesNotes = [];
        const scanNotes = [];

        const messageObj = getNodeScanMessages(nodesNotes, scanNotes);

        expect(messageObj).toEqual({});
    });

    it('should return an object for missingScanData when node notes contain MISSING_SCAN_DATA', () => {
        const nodesNotes = ['MISSING_SCAN_DATA'];
        const scanNotes = [];

        const messageObj = getNodeScanMessages(nodesNotes, scanNotes);

        expect(messageObj).toEqual(nodeScanMessages.missingScanData);
    });

    it('should return an object for osUnsupported when scan notes contain OS_UNSUPPORTED', () => {
        const nodesNotes = [];
        const scanNotes = ['OS_UNSUPPORTED'];

        const messageObj = getNodeScanMessages(nodesNotes, scanNotes);

        expect(messageObj).toEqual(nodeScanMessages.osUnsupported);
    });
});
