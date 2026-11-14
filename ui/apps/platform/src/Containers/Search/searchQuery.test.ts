import {
    parseQueryString,
    parseSearchFilter,
    stringifyQueryObject,
    stringifySearchFilter,
} from './searchQuery';

describe('parseQueryString', () => {
    it('keeps a key that is in the dynamic search options', () => {
        const { searchFilter } = parseQueryString('?s[CVE]=cve-1', ['CVE']);
        expect(searchFilter).toEqual({ CVE: 'cve-1' });
    });

    it('keeps a curated config search term even when dynamic options are empty', () => {
        // Preserves curated filtering if the search options request has not resolved or failed.
        const { searchFilter } = parseQueryString('?s[Deployment]=nginx', []);
        expect(searchFilter).toEqual({ Deployment: 'nginx' });
    });

    it('keeps an unresolved non-config term while search options are unavailable', () => {
        // During loading, or after the options request fails, search options are empty; preserve
        // bookmarked terms outside the curated config so they are applied, not silently dropped.
        const { searchFilter } = parseQueryString('?s[CVE]=cve-1', []);
        expect(searchFilter).toEqual({ CVE: 'cve-1' });
    });

    it('drops an unrecognized key once search options have loaded', () => {
        const { searchFilter } = parseQueryString('?s[Bogus]=x', ['CVE']);
        expect(searchFilter).toEqual({});
    });

    it('drops empty values', () => {
        const { searchFilter } = parseQueryString('?s[Deployment]=', []);
        expect(searchFilter).toEqual({});
    });

    it('preserves array values', () => {
        const { searchFilter } = parseQueryString('?s[Cluster]=a&s[Cluster]=b', []);
        expect(searchFilter).toEqual({ Cluster: ['a', 'b'] });
    });

    it('parses the nav category when a filter is present', () => {
        const { navCategory } = parseQueryString('?s[Deployment]=nginx&category=Deployments', []);
        expect(navCategory).toBe('DEPLOYMENTS');
    });

    it('returns SEARCH_UNSET when there is no filter', () => {
        const { navCategory } = parseQueryString('?category=Deployments', []);
        expect(navCategory).toBe('SEARCH_UNSET');
    });

    it('preserves the perPage parameter', () => {
        const { perPage } = parseQueryString('?s[Cluster]=remote&perPage=50', ['Cluster']);
        expect(perPage).toBe('50');
    });

    it('leaves perPage undefined when absent and ignores the page parameter', () => {
        const { perPage } = parseQueryString('?s[Cluster]=remote&page=3', ['Cluster']);
        expect(perPage).toBeUndefined();
    });
});

describe('stringifyQueryObject', () => {
    it('returns an empty string for an empty filter', () => {
        expect(stringifyQueryObject({ searchFilter: {}, navCategory: 'SEARCH_UNSET' })).toBe('');
    });

    it('omits the category for SEARCH_UNSET', () => {
        const queryString = stringifyQueryObject({
            searchFilter: { Deployment: 'nginx' },
            navCategory: 'SEARCH_UNSET',
        });
        expect(queryString).toContain('Deployment');
        expect(queryString).not.toContain('category');
    });

    it('includes the category when it is set', () => {
        const queryString = stringifyQueryObject({
            searchFilter: { Deployment: 'nginx' },
            navCategory: 'DEPLOYMENTS',
        });
        expect(queryString).toContain('category=Deployments');
    });

    it('round-trips with parseQueryString', () => {
        const searchFilter = { Deployment: 'nginx', Cluster: ['a', 'b'] };
        const queryString = stringifyQueryObject({ searchFilter, navCategory: 'DEPLOYMENTS' });
        const parsed = parseQueryString(queryString, []);
        expect(parsed.searchFilter).toEqual(searchFilter);
        expect(parsed.navCategory).toBe('DEPLOYMENTS');
    });

    it('carries the page size forward while omitting the page', () => {
        const queryString = stringifyQueryObject({
            searchFilter: { Cluster: 'remote' },
            navCategory: 'SEARCH_UNSET',
            perPage: '50',
        });
        expect(queryString).toContain('perPage=50');
        expect(queryString).not.toContain('page=');
    });

    it('omits perPage when it is not provided', () => {
        const queryString = stringifyQueryObject({
            searchFilter: { Cluster: 'remote' },
            navCategory: 'SEARCH_UNSET',
        });
        expect(queryString).not.toContain('perPage');
    });

    it('round-trips the page size, resetting the page', () => {
        const parsed = parseQueryString('?s[Cluster]=remote&perPage=50&page=3', ['Cluster']);
        const queryString = stringifyQueryObject(parsed);
        expect(queryString).toContain('perPage=50');
        expect(queryString).not.toContain('page=3');
    });
});

describe('stringifySearchFilter and parseSearchFilter', () => {
    it('round-trips a search filter', () => {
        const searchFilter = { Deployment: 'nginx', Cluster: ['a', 'b'] };
        expect(parseSearchFilter(stringifySearchFilter(searchFilter))).toEqual(searchFilter);
    });
});
