import { imageScanMessages } from 'messages/vulnMgmt.messages';
import getImageScanMessage from './getImageScanMessage';

describe('getImageScanMessage', () => {
    it('should return an empty object when there are no notes in the notes arrays', () => {
        const imagesNotes = [];
        const scanNotes = [];

        const messageObj = getImageScanMessage(imagesNotes, scanNotes);

        expect(messageObj).toEqual({});
    });

    it('should return an object for missingMetadata when image notes contain MISSING_METADATA', () => {
        const imagesNotes = ['MISSING_METADATA'];
        const scanNotes = [];

        const messageObj = getImageScanMessage(imagesNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.missingMetadata);
    });

    it('should return an object for missingScanData when image notes contain MISSING_SCAN_DATA', () => {
        const imagesNotes = ['MISSING_SCAN_DATA'];
        const scanNotes = [];

        const messageObj = getImageScanMessage(imagesNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.missingScanData);
    });

    it('should return an object for osUnavailable when scan notes contain OS_UNAVAILABLE', () => {
        const imagesNotes = [];
        const scanNotes = ['OS_UNAVAILABLE'];

        const messageObj = getImageScanMessage(imagesNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.osUnavailable);
    });

    it('should return an object for languageCvesUnavailable when scan notes contain PARTIAL_SCAN_DATA and LANGUAGE_CVES_UNAVAILABLE', () => {
        const imagesNotes = [];
        const scanNotes = ['PARTIAL_SCAN_DATA', 'LANGUAGE_CVES_UNAVAILABLE'];

        const messageObj = getImageScanMessage(imagesNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.languageCvesUnavailable);
    });

    it('should return an object for osCvesUnavailable when scan notes contain PARTIAL_SCAN_DATA and OS_CVES_UNAVAILABLE', () => {
        const imagesNotes = [];
        const scanNotes = ['PARTIAL_SCAN_DATA', 'OS_CVES_UNAVAILABLE'];

        const messageObj = getImageScanMessage(imagesNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.osCvesUnavailable);
    });

    it('should return an object for osCvesUnavailable when scan notes contain OS_CVES_STALE', () => {
        const imagesNotes = [];
        const scanNotes = ['OS_CVES_STALE'];

        const messageObj = getImageScanMessage(imagesNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.osCvesStale);
    });

    it('should return an object for certifiedRHELUnavailable when scan notes contain PARTIAL_SCAN_DATA and CERTIFIED_RHEL_SCAN_UNAVAILABLE', () => {
        const imagesNotes = [];
        const scanNotes = ['PARTIAL_SCAN_DATA', 'CERTIFIED_RHEL_SCAN_UNAVAILABLE'];

        const messageObj = getImageScanMessage(imagesNotes, scanNotes);

        expect(messageObj).toEqual(imageScanMessages.certifiedRHELUnavailable);
    });
});
