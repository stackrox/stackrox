import { getSbomGenerationStatusMessage } from './getSbomGenerationStatusMessage';

describe('getSbomGenerationStatusMessage', () => {
    it('should return undefined (enabled) when Scanner V4 is enabled and there are no notes', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: [],
                scanNotes: [],
            })
        ).toBeUndefined();
    });

    it('should return the Scanner V4 message when Scanner V4 is not enabled', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: false,
                imageNotes: [],
                scanNotes: [],
            })
        ).toBe('SBOM generation requires Scanner V4');
    });

    it('should prefer the Scanner V4 message even when scan data is missing', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: false,
                imageNotes: ['MISSING_SCAN_DATA'],
                scanNotes: [],
            })
        ).toBe('SBOM generation requires Scanner V4');
    });

    it('should block generation when image notes contain MISSING_SCAN_DATA', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: ['MISSING_SCAN_DATA'],
                scanNotes: [],
            })
        ).toBe('SBOM generation is unavailable due to incomplete scan data');
    });

    it('should block generation when image notes contain MISSING_METADATA', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: ['MISSING_METADATA'],
                scanNotes: [],
            })
        ).toBe('SBOM generation is unavailable due to incomplete scan data');
    });

    // The core of ROX-36762: an image with no detectable base OS (e.g. FROM scratch)
    // still has scan data, so SBOM generation must remain enabled even though the
    // "CVE data may be inaccurate" banner is shown.
    it('should NOT block generation when scan notes contain OS_UNAVAILABLE', () => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: [],
                scanNotes: ['OS_UNAVAILABLE'],
            })
        ).toBeUndefined();
    });

    it.each([
        ['PARTIAL_SCAN_DATA', 'OS_CVES_UNAVAILABLE'],
        ['PARTIAL_SCAN_DATA', 'LANGUAGE_CVES_UNAVAILABLE'],
        ['PARTIAL_SCAN_DATA', 'CERTIFIED_RHEL_SCAN_UNAVAILABLE'],
        ['OS_CVES_STALE'],
    ])('should NOT block generation for CVE-accuracy scan notes %j', (...scanNotes) => {
        expect(
            getSbomGenerationStatusMessage({
                isScannerV4Enabled: true,
                imageNotes: [],
                scanNotes,
            })
        ).toBeUndefined();
    });
});
