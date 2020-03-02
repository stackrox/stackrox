import { getViewStateFromSearch } from './searchUtils';

describe('getViewStateFromSearch', () => {
    it('should return false when passed an empty search object', () => {
        const searchObj = {};
        const key = 'CVE Suppressed';

        const containsKey = getViewStateFromSearch(searchObj, key);

        expect(containsKey).toEqual(false);
    });

    it('should return false when key is not in the given search object', () => {
        const searchObj = { CVE: 'CVE-2019-9893' };
        const key = 'CVE Suppressed';

        const containsKey = getViewStateFromSearch(searchObj, key);

        expect(containsKey).toEqual(false);
    });

    it('should return true when key is in the given search object', () => {
        const searchObj = { 'CVE Suppressed': true, CVE: 'CVE-2019-9893' };
        const key = 'CVE Suppressed';

        const containsKey = getViewStateFromSearch(searchObj, key);

        expect(containsKey).toEqual(true);
    });

    it('should return false when key is in the given search object but its value is false', () => {
        const searchObj = { 'CVE Suppressed': 'false' };
        const key = 'CVE Suppressed';

        const containsKey = getViewStateFromSearch(searchObj, key);

        expect(containsKey).toEqual(false);
    });

    it('should return false when key is in the given search object but its value is string "false"', () => {
        const searchObj = { 'CVE Suppressed': false };
        const key = 'CVE Suppressed';

        const containsKey = getViewStateFromSearch(searchObj, key);

        expect(containsKey).toEqual(false);
    });
});
