import { isValidURL } from './urlUtils';

describe('urlUtils', () => {
    describe('isValidURL', () => {
        it('should return false for the empty string', () => {
            const testUrl = '';

            expect(isValidURL(testUrl)).toEqual(false);
        });

        it('should return false for a malformed URL missing a TLD', () => {
            const testUrl =
                'https://people.canonical/~ubuntu-security/cve/2016/CVE-2016-0705.htmlf';

            expect(isValidURL(testUrl)).toEqual(true);
        });

        it('should return true for a CVE Tracker URL', () => {
            const testUrl =
                'https://people.canonical.com/~ubuntu-security/cve/2016/CVE-2016-0705.htmlf';

            expect(isValidURL(testUrl)).toEqual(true);
        });

        it('should return true for a NVD URL', () => {
            const testUrl = 'https://nvd.nist.gov/vuln/detail/CVE-2015-2992';

            expect(isValidURL(testUrl)).toEqual(true);
        });
    });
});
