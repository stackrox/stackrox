import Raven from 'raven-js';

import { imageScanMessages } from 'messages/vulnMgmt.messages';
import getImageScanMessage from './getImageScanMessage';

jest.mock('raven-js', () => ({
    captureException: jest.fn(),
}));

describe('getImageScanMessage', () => {
    it('should return an empty object when there are no notes in the notes arrays', () => {
        const imageNotes = [];
        const scanNotes = [];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        expect(messageObj).toEqual({});
    });

    it('should return an object for missingMetadata when image notes contain MISSING_METADATA', () => {
        const imageNotes = ['MISSING_METADATA'];
        const scanNotes = [];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.missingMetadata);
    });

    it('should return an object for missingScanData when image notes contain MISSING_SCAN_DATA', () => {
        const imageNotes = ['MISSING_SCAN_DATA'];
        const scanNotes = [];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.missingScanData);
    });

    it('should return an object for osUnavailable when scan notes contain OS_UNAVAILABLE', () => {
        const imageNotes = [];
        const scanNotes = ['OS_UNAVAILABLE'];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.osUnavailable);
    });

    it('should return an object for languageCvesUnavailable when scan notes contain PARTIAL_SCAN_DATA and LANGUAGE_CVES_UNAVAILABLE', () => {
        const imageNotes = [];
        const scanNotes = ['PARTIAL_SCAN_DATA', 'LANGUAGE_CVES_UNAVAILABLE'];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.languageCvesUnavailable);
    });

    it('should return an object for osCvesUnavailable when scan notes contain PARTIAL_SCAN_DATA and OS_CVES_UNAVAILABLE', () => {
        const imageNotes = [];
        const scanNotes = ['PARTIAL_SCAN_DATA', 'OS_CVES_UNAVAILABLE'];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.osCvesUnavailable);
    });

    it('should return an object for osCvesUnavailable when scan notes contain OS_CVES_STALE', () => {
        const imageNotes = [];
        const scanNotes = ['OS_CVES_STALE'];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.osCvesStale);
    });

    it('should return an object for certifiedRHELUnavailable when scan notes contain PARTIAL_SCAN_DATA and CERTIFIED_RHEL_SCAN_UNAVAILABLE', () => {
        const imageNotes = [];
        const scanNotes = ['PARTIAL_SCAN_DATA', 'CERTIFIED_RHEL_SCAN_UNAVAILABLE'];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.certifiedRHELUnavailable);
    });

    it('should capture the error when an unknown state is encountered', () => {
        // Spy on Raven.captureException
        const spy = jest.spyOn(Raven, 'captureException');

        const imageNotes = [];
        const scanNotes = ['THIS_IS_NOT_OK'];

        const messageObj = getImageScanMessage(imageNotes, scanNotes);

        // Assert that captureException was called with proper extra values
        expect(spy).toHaveBeenCalledWith(
            expect.any(Error),
            expect.objectContaining({
                extra: { imageNotes, scanNotes },
            })
        );

        // Assert that the error message is correct
        const errorArg = spy.mock.calls[0][0] as Error;
        expect(errorArg).toBeInstanceOf(Error);
        expect(errorArg.message).toBe('Unknown State Detected');

        expect(messageObj).toEqual({});

        // Restore the original function
        spy.mockRestore();
    });
});
