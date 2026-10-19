import { getSbomGenerationStatusMessage } from './getSbomGenerationStatusMessage';

describe('getSbomGenerationStatusMessage', () => {
    it('should return undefined (enabled) when Scanner V4 is enabled and there are no notes', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: [],
            })
        ).toBeUndefined();
    });

    it('should return the Scanner V4 message when Scanner V4 is not enabled', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: false,
                imageNotes: [],
            })
        ).toBe('SBOM generation requires Scanner V4');
    });

    it('should prefer the Scanner V4 message even when scan data is missing', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: false,
                imageNotes: ['MISSING_SCAN_DATA'],
            })
        ).toBe('SBOM generation requires Scanner V4');
    });

    it('should block generation when image notes contain MISSING_SCAN_DATA', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: ['MISSING_SCAN_DATA'],
            })
        ).toBe('SBOM generation is unavailable due to incomplete scan data');
    });

    it('should block generation when image notes contain MISSING_METADATA', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: ['MISSING_METADATA'],
            })
        ).toBe('SBOM generation is unavailable due to incomplete scan data');
    });

    // The core of ROX-36762: an image with no detectable base OS (e.g. FROM scratch)
    // has scan data but only scan-level notes (OS_UNAVAILABLE etc.), never the
    // image-level MISSING_* notes. Those scan-level notes must not block generation,
    // so with empty imageNotes the function keeps SBOM generation enabled.
    it('should NOT block generation when the image has no missing-data notes', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: [],
            })
        ).toBeUndefined();
    });
});
