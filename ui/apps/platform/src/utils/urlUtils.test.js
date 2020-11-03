import { isValidURL, isValidCidrBlock } from './urlUtils';

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

    describe('isValidCidrBlock', () => {
        it('should return false for the empty string', () => {
            const testUrl = '';

            expect(isValidURL(testUrl)).toEqual(false);
        });

        it('should return true for valid IPv4 CIDR block', () => {
            const testUrl = '192.168.0.1/7';

            expect(isValidCidrBlock(testUrl)).toEqual(true);
        });

        it('should return false for invalid IPv4 CIDR block', () => {
            const testUrl = '192.168.0.1/33';

            expect(isValidCidrBlock(testUrl)).toEqual(false);
        });

        it('should return true for valid full IPv6 CIDR block', () => {
            const testUrl = '2002::1234:abcd:ffff:c0a8:101/64';

            expect(isValidCidrBlock(testUrl)).toEqual(true);
        });

        it('should return true for partial IPv6 CIDR block', () => {
            const testUrl = '2001:c00::/23';

            expect(isValidCidrBlock(testUrl)).toEqual(true);
        });

        it('should return true for short IPv6 CIDR block', () => {
            const testUrl = '::ffff:0:0/8';

            expect(isValidCidrBlock(testUrl)).toEqual(true);
        });

        it('should return false for invalid IPv6 CIDR block', () => {
            const testUrl = '1200::AB00:1234::2552:7777:1313';

            expect(isValidCidrBlock(testUrl)).toEqual(false);
        });
    });
});
