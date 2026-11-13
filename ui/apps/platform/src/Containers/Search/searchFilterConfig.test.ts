import { getUnconfiguredChipAttributes, searchFilterConfig } from './searchFilterConfig';

describe('searchFilterConfig', () => {
    it('includes one entity per global search result category', () => {
        expect(searchFilterConfig).toHaveLength(13);
    });

    it('has a unique display name for every entity', () => {
        const displayNames = searchFilterConfig.map((entity) => entity.displayName);
        expect(new Set(displayNames).size).toBe(displayNames.length);
    });

    it('has a unique search category for every entity', () => {
        const searchCategories = searchFilterConfig.map((entity) => entity.searchCategory);
        expect(new Set(searchCategories).size).toBe(searchCategories.length);
    });

    it('gives every entity at least one attribute', () => {
        searchFilterConfig.forEach((entity) => {
            expect(entity.attributes.length).toBeGreaterThan(0);
        });
    });

    // Regression guard: CompoundSearchFilterLabels renders the chip row by flattening the
    // attributes of every entity, so a search term shared by two entities renders a duplicate
    // chip (this is what caused "Violation time" to appear twice).
    it('does not reuse a search term across entities', () => {
        const searchTerms = searchFilterConfig.flatMap((entity) =>
            entity.attributes.map((attribute) => attribute.searchTerm)
        );
        const duplicates = searchTerms.filter((term, index) => searchTerms.indexOf(term) !== index);
        expect(duplicates).toEqual([]);
    });

    it('exposes the Violation entity with a human-readable Time attribute', () => {
        const violation = searchFilterConfig.find((entity) => entity.displayName === 'Violation');
        expect(violation).toBeDefined();

        const timeAttribute = violation?.attributes.find(
            (attribute) => attribute.searchTerm === 'Violation Time'
        );
        expect(timeAttribute?.displayName).toBe('Time');
    });

    it('includes a Deployment entity to serve as the default', () => {
        expect(searchFilterConfig.some((entity) => entity.displayName === 'Deployment')).toBe(true);
    });

    // Legacy global search offered CVE, Component, and Process Name as flat terms. They are
    // restored under a single entity each (CVE/Component under Image, Process Name under
    // Deployment) so the flat global query still matches every result category without the
    // duplicate chip that reusing a term across entities would produce.
    it('restores the legacy CVE, Component, and Process Name terms exactly once', () => {
        const searchTerms = searchFilterConfig.flatMap((entity) =>
            entity.attributes.map((attribute) => attribute.searchTerm)
        );
        ['CVE', 'Component', 'Process Name'].forEach((term) => {
            expect(searchTerms.filter((searchTerm) => searchTerm === term)).toHaveLength(1);
        });
    });
});

describe('getUnconfiguredChipAttributes', () => {
    // Safety net: when the dynamic search options are unavailable, parseQueryString keeps every
    // bookmarked key; any key this config does not expose must still render as a removable text chip
    // rather than an invisible filter. (A key the backend does not recognize as an option, such as
    // the internal "Priority" field, is dropped upstream by parseQueryString and never reaches here.)
    it('builds a removable text attribute for an active key the config does not expose', () => {
        expect(
            getUnconfiguredChipAttributes({ 'Legacy Field': ['x'] }, searchFilterConfig)
        ).toEqual([
            {
                displayName: 'Legacy Field',
                filterChipLabel: 'Legacy Field',
                searchTerm: 'Legacy Field',
                inputType: 'text',
            },
        ]);
    });

    it('ignores keys covered by the config, case-insensitively, so no duplicate chip renders', () => {
        expect(
            getUnconfiguredChipAttributes({ CVE: ['x'], cvss: ['>1'] }, searchFilterConfig)
        ).toEqual([]);
    });

    it('returns nothing for an empty filter', () => {
        expect(getUnconfiguredChipAttributes({}, searchFilterConfig)).toEqual([]);
    });
});
