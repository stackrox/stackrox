import { parseQueryString, stringifyQueryObject } from './searchQuery';

const searchOptions = ['Cluster', 'Namespace'];

describe('parseQueryString', () => {
    it('should parse search filter and leave navCategory unset without a category', () => {
        const { searchFilter, navCategory, perPage } = parseQueryString(
            '?s[Cluster]=remote',
            searchOptions
        );

        expect(searchFilter).toEqual({ Cluster: 'remote' });
        expect(navCategory).toBe('SEARCH_UNSET');
        expect(perPage).toBeUndefined();
    });

    it('should preserve the perPage parameter', () => {
        const { perPage } = parseQueryString('?s[Cluster]=remote&perPage=50', searchOptions);

        expect(perPage).toBe('50');
    });

    it('should not treat the page parameter as perPage', () => {
        const { perPage } = parseQueryString('?s[Cluster]=remote&page=3', searchOptions);

        expect(perPage).toBeUndefined();
    });
});

describe('stringifyQueryObject', () => {
    it('should return an empty string for an empty search filter', () => {
        expect(stringifyQueryObject({ searchFilter: {}, navCategory: 'SEARCH_UNSET' })).toBe('');
    });

    it('should carry the page size forward while omitting the page', () => {
        const queryString = stringifyQueryObject({
            searchFilter: { Cluster: 'remote' },
            navCategory: 'SEARCH_UNSET',
            perPage: '50',
        });

        expect(queryString).toContain('perPage=50');
        expect(queryString).not.toContain('page=');
    });

    it('should omit perPage when it is not provided', () => {
        const queryString = stringifyQueryObject({
            searchFilter: { Cluster: 'remote' },
            navCategory: 'SEARCH_UNSET',
        });

        expect(queryString).not.toContain('perPage');
    });

    it('should round-trip the page size through parse and stringify', () => {
        const parsed = parseQueryString('?s[Cluster]=remote&perPage=50&page=3', searchOptions);
        const queryString = stringifyQueryObject(parsed);

        // Page size survives, page resets (is dropped).
        expect(queryString).toContain('perPage=50');
        expect(queryString).not.toContain('page=3');
    });
});
