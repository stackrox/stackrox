import { doesSearchContain } from './searchUtils';

describe('doesSearchContain', () => {
    it('should return false when passed an empty search object', () => {
        const searchObj = {};
        const key = 'CVE Suppressed';

        const containsKey = doesSearchContain(searchObj, key);

        expect(containsKey).toEqual(false);
    });

    it('should return false when key is not in the given search object', () => {
        const searchObj = { CVE: 'CVE-2019-9893' };
        const key = 'CVE Suppressed';

        const containsKey = doesSearchContain(searchObj, key);

        expect(containsKey).toEqual(false);
    });

    it('should return true when key is in the given search object', () => {
        const searchObj = { 'CVE Suppressed': true, CVE: 'CVE-2019-9893' };
        const key = 'CVE Suppressed';

        const containsKey = doesSearchContain(searchObj, key);

        expect(containsKey).toEqual(true);
    });
});
